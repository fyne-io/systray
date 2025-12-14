package mock

import (
	"errors"
	"fmt"
	"log"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/godbus/dbus/v5"
)

const ()

type SignalInfo struct {
	Interface string
	Member    string
	Path      string
}

func newSignalInfo(iface, member, path string) SignalInfo {
	if !strings.HasPrefix(iface, "org.") {
		panic("invalid interface name")
	}
	if path != "" && !strings.HasPrefix(path, "/") {
		panic("invalid object path")
	}

	return SignalInfo{
		Interface: iface,
		Member:    member,
		Path:      path,
	}
}

func (s SignalInfo) Name() string {
	return fmt.Sprintf("%s.%s", s.Interface, s.Member)
}

var (
	itemSignals = []SignalInfo{
		// newSignalInfo("org.kde.StatusNotifierItem", "NewIcon", "/StatusNotifierItem"),
		// newSignalInfo("org.kde.StatusNotifierItem", "NewAttentionIcon", "/StatusNotifierItem"),
		// newSignalInfo("org.kde.StatusNotifierItem", "NewOverlayIcon", "/StatusNotifierItem"),
		// newSignalInfo("org.kde.StatusNotifierItem", "NewToolTip", "/StatusNotifierItem"),
		// newSignalInfo("org.kde.StatusNotifierItem", "NewStatus", "/StatusNotifierItem"),
		newSignalInfo("org.kde.StatusNotifierItem", "NewMenu", "/StatusNotifierItem"),
		// newSignalInfo("org.kde.StatusNotifierItem", "NewTitle", "/StatusNotifierItem"),
		// newSignalInfo("org.kde.StatusNotifierItem", "NewIconThemePath", "/StatusNotifierItem"),
	}

	menuSignals = []SignalInfo{
		newSignalInfo("org.kde.StatusNotifierMenu", "LayoutUpdated", "/StatusNotifierMenu"),
		newSignalInfo("org.kde.StatusNotifierMenu", "ItemActivationRequested", "/StatusNotifierMenu"),
		newSignalInfo("org.kde.StatusNotifierMenu", "ItemsPropertiesUpdated", "/StatusNotifierMenu"),
	}

	itemPropChangeSignal = newSignalInfo("org.freedesktop.DBus.Properties", "PropertiesChanged", "/StatusNotifierItem")
	nameChangedSignal    = newSignalInfo("org.freedesktop.DBus", "NameOwnerChanged", "")
)

// StatusNotifierWatcher simulates the system tray daemon
type StatusNotifierWatcher struct {
	ItemRegisteredCh chan *StatusNotifierItem
	ItemRemoveCh     chan *StatusNotifierItem
	CloseCh          chan struct{}
	conn             *dbus.Conn
	daemon           *DbusDaemon
	mu               sync.Mutex
	items            map[dbus.Sender]*StatusNotifierItem
	signalsCh        chan *dbus.Signal
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
		ItemRegisteredCh: make(chan *StatusNotifierItem, 1),
		ItemRemoveCh:     make(chan *StatusNotifierItem, 1),
		CloseCh:          make(chan struct{}),
		conn:             conn,
		daemon:           daemon,
		items:            make(map[dbus.Sender]*StatusNotifierItem),
		signalsCh:        make(chan *dbus.Signal, 10),
	}

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
	w.conn.Signal(w.signalsCh)

	// Start goroutine to handle dbus signals
	go w.handleSignals(w.signalsCh)

	// register name change notifications
	w.addOrRemoveSignal(true, "", nameChangedSignal)

	fmt.Println("StatusNotifierWatcher service started")
	return w
}

// CleanupMockDBusWatcher releases the D-Bus name and closes the connection
func (w *StatusNotifierWatcher) Close() {
	if w.CloseCh == nil {
		return
	}
	log.Println("Close channel")
	w.conn.RemoveSignal(w.signalsCh)
	close(w.signalsCh)

	close(w.ItemRegisteredCh)
	close(w.ItemRemoveCh)

	if w.conn != nil {
		w.conn.ReleaseName("org.kde.StatusNotifierWatcher")
		w.conn.Close()
		w.conn = nil
	}

	if w.daemon != nil {
		w.daemon.Close()
		w.daemon = nil
	}

	close(w.CloseCh)
	w.CloseCh = nil
}

// ===================================================================
// StatusNotifierWatcher dbus methods
// ===================================================================

// org.kde.StatusNotifierWatcher.RegisterStatusNotifierItem
func (w *StatusNotifierWatcher) RegisterStatusNotifierItem(sender dbus.Sender, service string) *dbus.Error {
	log.Println("Register item:", service, "from sender:", string(sender))

	item := w.addSni(sender, service)
	w.registerItemSignals(sender)
	w.fetchItemProperties(sender)

	w.ItemRegisteredCh <- item

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

// func (w *StatusNotifierWatcher) Items() []StatusNotifierItem {
// 	w.mu.Lock()
// 	defer w.mu.Unlock()
// 	items := make([]StatusNotifierItem, 0, len(w.items))
// 	for _, item := range w.items {
// 		items = append(items, *item)
// 	}
// 	return items
// }

func (w *StatusNotifierWatcher) addSni(sender dbus.Sender, service string) *StatusNotifierItem {
	w.mu.Lock()
	defer w.mu.Unlock()

	if item, exists := w.items[sender]; exists {
		log.Println("Item already registered, ignoring")
		return item
	}

	item := newStatusNotifierItem(service, sender)
	w.items[sender] = item

	return item
}

func (w *StatusNotifierWatcher) fetchItemProperties(sender dbus.Sender) {
	senderName := string(sender)
	w.mu.Lock()
	item, exists := w.items[sender]
	if !exists {
		w.mu.Unlock()
		log.Println("Item not found for sender:", senderName)
		return
	}
	w.mu.Unlock()

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

	if err := ParseItemProperties(item.Properties, props); err != nil {
		log.Println("Failed to parse properties:", err)
		return
	}

	log.Println("Updated properties for sender:", senderName)
}

func (w *StatusNotifierWatcher) propertyChanged(sender dbus.Sender, changedProps map[string]dbus.Variant) {
	senderName := string(sender)
	w.mu.Lock()
	item, exists := w.items[sender]
	if !exists {
		w.mu.Unlock()
		log.Println("Item not found for sender:", senderName)
		return
	}
	w.mu.Unlock()

	log.Println("Processing property changes for sender:", senderName)

	if err := ParseItemProperties(item.Properties, changedProps); err != nil {
		log.Println("Failed to update properties:", err)
		return
	}

	log.Println("Updated properties for sender:", senderName)
}

func (w *StatusNotifierWatcher) addOrRemoveSignal(add bool, sender dbus.Sender, signal SignalInfo) error {

	matchRule := fmt.Sprintf("type='signal',interface='%s'", signal.Interface)
	if sender != "" {
		matchRule += fmt.Sprintf(",sender='%s'", string(sender))
	}
	if signal.Member != "" {
		matchRule += fmt.Sprintf(",member='%s'", signal.Member)
	}
	if signal.Path != "" {
		matchRule += fmt.Sprintf(",path='%s'", signal.Path)
	}

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

func (w *StatusNotifierWatcher) registerItemSignals(sender dbus.Sender) error {
	// register item signals
	for _, signal := range itemSignals {
		if err := w.addOrRemoveSignal(true, sender, signal); err != nil {
			return fmt.Errorf("failed to register signal %s: %w", signal, err)
		}
	}

	// register menu signals
	for _, signal := range menuSignals {
		if err := w.addOrRemoveSignal(true, sender, signal); err != nil {
			return fmt.Errorf("failed to register signal %s: %w", signal, err)
		}
	}

	if err := w.addOrRemoveSignal(true, sender, itemPropChangeSignal); err != nil {
		return fmt.Errorf("failed to register signal %s: %w", itemPropChangeSignal, err)
	}

	return nil
}

// handleSignals processes incoming D-Bus signals
func (w *StatusNotifierWatcher) handleSignals(signals chan *dbus.Signal) {
	splitSignalName := func(signalName string) (string, string) {
		i := strings.LastIndex(signalName, ".")
		ifName := signalName[:i]
		method := signalName[i+1:]
		return ifName, method
	}
	for signal := range signals {
		log.Printf("Received signal: %+v\n", signal)

		ifName, method := splitSignalName(signal.Name)
		sender := dbus.Sender(signal.Sender)

		if ifName == "org.kde.StatusNotifierItem" && slices.Contains(itemSignals, SignalInfo{ifName, method, "/StatusNotifierItem"}) {
			w.fetchItemProperties(sender)
			return
		}

		switch signal.Name {
		case nameChangedSignal.Name():
			w.onNameOwnerChanged(signal.Body[1].(string), signal.Body[2].(string))

		case itemPropChangeSignal.Name():
			if len(signal.Body) < 2 || signal.Body[0].(string) != "org.kde.StatusNotifierItem" {
				log.Println("Ignore event")
				return
			}
			w.propertyChanged(sender, signal.Body[1].(map[string]dbus.Variant))
		}
	}
}

func (w *StatusNotifierWatcher) onNameOwnerChanged(old, new string) {
	if new != "" {
		return // not a disappearance
	}

	sender := dbus.Sender(old)

	w.mu.Lock()
	item, exists := w.items[sender]
	if !exists {
		w.mu.Unlock()
		return
	}

	delete(w.items, sender)
	w.mu.Unlock()

	w.ItemRemoveCh <- item
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
