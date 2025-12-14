package mock

import (
	"errors"
	"fmt"
	"log"
	"slices"
	"sync"
	"testing"

	"github.com/godbus/dbus/v5"
)

var (
	itemSignals = []string{
		"NewIcon",
		"NewAttentionIcon",
		"NewOverlayIcon",
		"NewToolTip",
		"NewStatus",
		"NewMenu",
	}

	menuSignals = []string{
		"LayoutUpdated",
		"ItemActivationRequested",
		"ItemsPropertiesUpdated",
	}
)

// StatusNotifierWatcher simulates the system tray daemon
type StatusNotifierWatcher struct {
	conn   *dbus.Conn
	daemon *DbusDaemon
	mu     sync.Mutex
	items  map[dbus.Sender]StatusNotifierItem
}

func NewStatusNotifierWatcher(t *testing.T) *StatusNotifierWatcher {
	daemon := startDbusDaemon(t)

	conn, err := createDBusConnection()
	if err != nil {
		t.Fatalf("failed to connect to D-Bus: %s", err)
	}

	reply, err := conn.RequestName("org.kde.StatusNotifierWatcher", dbus.NameFlagDoNotQueue)
	if err != nil {
		t.Fatalf("failed to request name: %s", err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		t.Fatalf("name already taken")
	}

	w := &StatusNotifierWatcher{
		conn:   conn,
		daemon: daemon,
		items:  make(map[dbus.Sender]StatusNotifierItem),
	}

	// Setup signal handler to capture sender information
	if err = conn.Export(w,
		"/StatusNotifierWatcher",
		"org.kde.StatusNotifierWatcher",
	); err != nil {
		t.Fatalf("failed to export object: %s", err)
	}

	// Export Properties interface
	if err := conn.Export(
		w,
		"/StatusNotifierWatcher",
		"org.freedesktop.DBus.Properties",
	); err != nil {
		t.Fatalf("failed to export properties interface: %s", err)
	}

	// Create signal channel
	signals := make(chan *dbus.Signal, 10)
	w.conn.Signal(signals)

	// Start goroutine to handle dbus signals
	go w.handleSignals(signals)

	fmt.Println("StatusNotifierWatcher service started")
	return w
}

// CleanupMockDBusWatcher releases the D-Bus name and closes the connection
func (w *StatusNotifierWatcher) Close() {
	if w.conn != nil {
		w.conn.ReleaseName("org.kde.StatusNotifierWatcher")
		w.conn.Close()
		w.conn = nil
	}

	if w.daemon != nil {
		w.daemon.Close()
		w.daemon = nil
	}
}

// ===================================================================
// StatusNotifierWatcher dbus methods
// ===================================================================

// org.kde.StatusNotifierWatcher.RegisterStatusNotifierItem
func (w *StatusNotifierWatcher) RegisterStatusNotifierItem(sender dbus.Sender, service string) *dbus.Error {
	senderName := string(sender)
	log.Println("Register item:", service, "from sender:", senderName)

	w.addSni(sender, service)
	w.registerSignals(sender)

	w.fetchItemProperties(sender)
	return nil
}

// org.kde.StatusNotifierWatcher.RegisterStatusNotifierHost
func (w *StatusNotifierWatcher) RegisterStatusNotifierHost(service string) *dbus.Error {
	log.Println("Register host:", service)
	return nil
}

// org.freedesktop.DBus.Properties.Get
func (w *StatusNotifierWatcher) Get(iface, prop string) (dbus.Variant, *dbus.Error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if iface != "org.kde.StatusNotifierWatcher" {
		return dbus.Variant{}, dbus.MakeFailedError(errors.New("wrong interface"))
	}

	switch prop {
	case "RegisteredStatusNotifierItems":
		return dbus.MakeVariant(w.items), nil
	case "IsStatusNotifierHostRegistered":
		return dbus.MakeVariant(true), nil
	}

	return dbus.Variant{}, dbus.MakeFailedError(fmt.Errorf("unknown property %s", prop))
}

// org.freedesktop.DBus.Properties.GetAll
func (w *StatusNotifierWatcher) GetAll(iface string) (map[string]dbus.Variant, *dbus.Error) {
	if iface != "org.kde.StatusNotifierWatcher" {
		return nil, dbus.MakeFailedError(errors.New("wrong interface"))
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	return map[string]dbus.Variant{
		"RegisteredStatusNotifierItems":  dbus.MakeVariant(w.items),
		"IsStatusNotifierHostRegistered": dbus.MakeVariant(true),
	}, nil
}

func (w *StatusNotifierWatcher) Items() []StatusNotifierItem {
	w.mu.Lock()
	defer w.mu.Unlock()
	items := make([]StatusNotifierItem, 0, len(w.items))
	for _, item := range w.items {
		items = append(items, item)
	}
	return items
}

func (w *StatusNotifierWatcher) addSni(sender dbus.Sender, service string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, exists := w.items[sender]; exists {
		log.Println("Item already registered, ignoring")
		return
	}

	w.items[sender] = StatusNotifierItem{
		Service: service,
		Sender:  sender,
	}
}

func (w *StatusNotifierWatcher) fetchItemProperties(sender dbus.Sender) {
	senderName := string(sender)
	log.Println("Fetching properties for sender:", senderName)

	obj := w.conn.Object(
		senderName,
		"/StatusNotifierItem",
	)
	call := obj.Call(
		"org.freedesktop.DBus.Properties.GetAll",
		0,
		"org.kde.StatusNotifierItem",
	)

	if call.Err != nil {
		log.Println("Call failed:", call.Err)
		return
	}

	props, ok := call.Body[0].(map[string]dbus.Variant)
	if !ok {
		log.Println("Invalid properties format")
		return
	}

	properties, err := ParseItemProperties(senderName, props)
	if err != nil {
		log.Println("Failed to parse properties:", err)
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	item, exists := w.items[sender]
	if !exists {
		log.Println("Item not found for sender:", senderName)
		return
	}

	item.Properties = properties
	w.items[sender] = item

	log.Println("Updated properties for sender:", senderName)
}

func (w *StatusNotifierWatcher) addRemoveSignal(add bool, sender dbus.Sender, member string, path string, interfaceName string) error {
	matchRule := fmt.Sprintf("type='signal',interface='%s',path='%s',member='%s',sender='%s'",
		interfaceName, path, member, string(sender))
	method := "AddMatch"
	if !add {
		method = "RemoveMatch"
	}

	err := w.conn.BusObject().Call(fmt.Sprintf("org.freedesktop.DBus.%s", method), 0, matchRule).Err
	if err != nil {
		return fmt.Errorf("failed to add match rule: %w", err)
	}
	return nil
}

func (w *StatusNotifierWatcher) registerSignals(sender dbus.Sender) error {
	// register item signals
	for _, signal := range itemSignals {
		if err := w.addRemoveSignal(true, sender, signal, "/StatusNotifierItem", "org.kde.StatusNotifierItem"); err != nil {
			return fmt.Errorf("failed to register signal %s: %w", signal, err)
		}
	}

	// register menu signals
	for _, signal := range menuSignals {
		if err := w.addRemoveSignal(true, sender, signal, "/StatusNotifierMenu", "com.canonical.dbusmenu"); err != nil {
			return fmt.Errorf("failed to register signal %s: %w", signal, err)
		}
	}

	return nil
}

// handleSignals processes incoming D-Bus signals
func (w *StatusNotifierWatcher) handleSignals(signals chan *dbus.Signal) {
	for signal := range signals {
		if slices.Contains(itemSignals, signal.Name) {
			log.Printf("Received signal: %s from %s\n", signal.Name, signal.Sender)

			w.fetchItemProperties(dbus.Sender(signal.Sender))
		}
		// if signal.Name == "org.kde.StatusNotifierWatcher.StatusNotifierItemRegistered" {
		// 	if len(signal.Body) > 0 {
		// 		service, ok := signal.Body[0].(string)
		// 		if !ok {
		// 			continue
		// 		}

		// 		sender := signal.Sender
		// 		fmt.Printf("Signal received: service=%s, sender=%s\n", service, sender)

		// 		// If service is a path, store the sender mapping
		// 		if strings.HasPrefix(service, "/") {
		// 			if m.itemSenders == nil {
		// 				m.itemSenders = make(map[string]string)
		// 			}
		// 			m.itemSenders[service] = sender
		// 			fmt.Printf("  Mapped path %s -> sender %s\n", service, sender)

		// 			// Now re-query with the correct sender
		// 			go m.queryItemProperties(service, sender)
		// 		}
		// 	}
		// }
	}
}

// func (w *MockStatusNotifierWatcher) emitItemsChanged() {
// 	w.conn.Emit(
// 		"/StatusNotifierWatcher",
// 		"org.freedesktop.DBus.Properties.PropertiesChanged",
// 		"org.kde.StatusNotifierWatcher",
// 		map[string]dbus.Variant{
// 			"RegisteredStatusNotifierItems": dbus.MakeVariant(w.items),
// 		},
// 		[]string{},
// 	)
// }
