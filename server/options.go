package server

import (
	"errors"
)

type Options struct {
	Address             string           `json:"address" flagName:"address" flagSName:"a" flagDescribe:"IP address to listen" default:"0.0.0.0"`
	Port                string           `json:"port" flagName:"port" flagSName:"p" flagDescribe:"Port number to liten" default:"8080"`
	PermitWrite         bool             `json:"permit_write" flagName:"permit-write" flagSName:"w" flagDescribe:"Permit clients to write to the TTY (BE CAREFUL)" default:"false"`
	EnableBasicAuth     bool             `json:"enable_basic_auth" default:"false"`
	Credential          string           `json:"credential" flagName:"credential" flagSName:"c" flagDescribe:"Credential for Basic Authentication (ex: user:pass, default disabled)" default:""`
	EnableRandomUrl     bool             `json:"enable_random_url" flagName:"random-url" flagSName:"r" flagDescribe:"Add a random string to the URL" default:"false"`
	RandomUrlLength     int              `json:"random_url_length" flagName:"random-url-length" flagDescribe:"Random URL length" default:"8"`
	EnableTLS           bool             `json:"enable_tls" flagName:"tls" flagSName:"t" flagDescribe:"Enable TLS/SSL" default:"false"`
	TLSCrtFile          string           `json:"tls_crt_file" flagName:"tls-crt" flagDescribe:"TLS/SSL certificate file path" default:"~/.gotty.crt"`
	TLSKeyFile          string           `json:"tls_key_file" flagName:"tls-key" flagDescribe:"TLS/SSL key file path" default:"~/.gotty.key"`
	EnableTLSClientAuth bool             `json:"enable_tls_client_auth" default:"false"`
	TLSCACrtFile        string           `json:"tls_ca_crt_file" flagName:"tls-ca-crt" flagDescribe:"TLS/SSL CA certificate file for client certifications" default:"~/.gotty.ca.crt"`
	IndexFile           string           `json:"index_file" flagName:"index" flagDescribe:"Custom index.html file" default:""`
	TitleFormat         string           `json:"title_format" flagName:"title-format" flagSName:"" flagDescribe:"Title format of browser window" default:"{{ .command }}@{{ .hostname }}"`
	EnableReconnect     bool             `json:"enable_reconnect" flagName:"reconnect" flagDescribe:"Enable reconnection" default:"false"`
	ReconnectTime       int              `json:"reconnect_time" flagName:"reconnect-time" flagDescribe:"Time to reconnect" default:"10"`
	MaxConnection       int              `json:"max_connection" flagName:"max-connection" flagDescribe:"Maximum connection to gotty" default:"0"`
	Once                bool             `json:"once" flagName:"once" flagDescribe:"Accept only one client and exit on disconnection" default:"false"`
	Timeout             int              `json:"timeout" flagName:"timeout" flagDescribe:"Timeout seconds for waiting a client(0 to disable)" default:"0"`
	PermitArguments     bool             `json:"permit_arguments" flagName:"permit-arguments" flagDescribe:"Permit clients to send command line arguments in URL (e.g. http://example.com:8080/?arg=AAA&arg=BBB)" default:"false"`
	Preferences         *HtermPrefernces `json:"preferences"`
	Width               int              `json:"width" flagName:"width" flagDescribe:"Static width of the screen, 0(default) means dynamically resize" default:"0"`
	Height              int              `json:"height" flagName:"height" flagDescribe:"Static height of the screen, 0(default) means dynamically resize" default:"0"`
	WSOrigin            string           `json:"ws_origin" flagName:"ws-origin" flagDescribe:"A regular expression that matches origin URLs to be accepted by WebSocket. No cross origin requests are acceptable by default" default:""`
	Term                string           `json:"term" flagName:"term" flagDescribe:"Terminal name to use on the browser, one of xterm or hterm." default:"xterm"`
	Debug               bool             `json:"debug" flagName:"debug" flagSName:"d" flagDescribe:"Enable debug logging for iPad/proxy/WebSocket diagnostics" default:"false"`

	TitleVariables map[string]interface{}
}

func (options *Options) Validate() error {
	if options.EnableTLSClientAuth && !options.EnableTLS {
		return errors.New("TLS client authentication is enabled, but TLS is not enabled")
	}
	return nil
}

type HtermPrefernces struct {
	AltGrMode                     *string                      `json:"alt-gr-mode,omitempty"`
	AltBackspaceIsMetaBackspace   bool                         `json:"alt-backspace-is-meta-backspace,omitempty"`
	AltIsMeta                     bool                         `json:"alt-is-meta,omitempty"`
	AltSendsWhat                  string                       `json:"alt-sends-what,omitempty"`
	AudibleBellSound              string                       `json:"audible-bell-sound,omitempty"`
	DesktopNotificationBell       bool                         `json:"desktop-notification-bell,omitempty"`
	BackgroundColor               string                       `json:"background-color,omitempty"`
	BackgroundImage               string                       `json:"background-image,omitempty"`
	BackgroundSize                string                       `json:"background-size,omitempty"`
	BackgroundPosition            string                       `json:"background-position,omitempty"`
	BackspaceSendsBackspace       bool                         `json:"backspace-sends-backspace,omitempty"`
	CharacterMapOverrides         map[string]map[string]string `json:"character-map-overrides,omitempty"`
	CloseOnExit                   bool                         `json:"close-on-exit,omitempty"`
	CursorBlink                   bool                         `json:"cursor-blink,omitempty"`
	CursorBlinkCycle              [2]int                       `json:"cursor-blink-cycle,omitempty"`
	CursorColor                   string                       `json:"cursor-color,omitempty"`
	ColorPaletteOverrides         []*string                    `json:"color-palette-overrides,omitempty"`
	CopyOnSelect                  bool                         `json:"copy-on-select,omitempty"`
	UseDefaultWindowCopy          bool                         `json:"use-default-window-copy,omitempty"`
	ClearSelectionAfterCopy       bool                         `json:"clear-selection-after-copy,omitempty"`
	CtrlPlusMinusZeroZoom         bool                         `json:"ctrl-plus-minus-zero-zoom,omitempty"`
	CtrlCCopy                     bool                         `json:"ctrl-c-copy,omitempty"`
	CtrlVPaste                    bool                         `json:"ctrl-v-paste,omitempty"`
	EastAsianAmbiguousAsTwoColumn bool                         `json:"east-asian-ambiguous-as-two-column,omitempty"`
	Enable8BitControl             *bool                        `json:"enable-8-bit-control,omitempty"`
	EnableBold                    *bool                        `json:"enable-bold,omitempty"`
	EnableBoldAsBright            bool                         `json:"enable-bold-as-bright,omitempty"`
	EnableClipboardNotice         bool                         `json:"enable-clipboard-notice,omitempty"`
	EnableClipboardWrite          bool                         `json:"enable-clipboard-write,omitempty"`
	EnableDec12                   bool                         `json:"enable-dec12,omitempty"`
	Environment                   map[string]string            `json:"environment,omitempty"`
	FontFamily                    string                       `json:"font-family,omitempty"`
	FontSize                      int                          `json:"font-size,omitempty"`
	FontSmoothing                 string                       `json:"font-smoothing,omitempty"`
	ForegroundColor               string                       `json:"foreground-color,omitempty"`
	HomeKeysScroll                bool                         `json:"home-keys-scroll,omitempty"`
	Keybindings                   map[string]string            `json:"keybindings,omitempty"`
	MaxStringSequence             int                          `json:"max-string-sequence,omitempty"`
	MediaKeysAreFkeys             bool                         `json:"media-keys-are-fkeys,omitempty"`
	MetaSendsEscape               bool                         `json:"meta-sends-escape,omitempty"`
	MousePasteButton              *int                         `json:"mouse-paste-button,omitempty"`
	PageKeysScroll                bool                         `json:"page-keys-scroll,omitempty"`
	PassAltNumber                 *bool                        `json:"pass-alt-number,omitempty"`
	PassCtrlNumber                *bool                        `json:"pass-ctrl-number,omitempty"`
	PassMetaNumber                *bool                        `json:"pass-meta-number,omitempty"`
	PassMetaV                     bool                         `json:"pass-meta-v,omitempty"`
	ReceiveEncoding               string                       `json:"receive-encoding,omitempty"`
	ScrollOnKeystroke             bool                         `json:"scroll-on-keystroke,omitempty"`
	ScrollOnOutput                bool                         `json:"scroll-on-output,omitempty"`
	ScrollbarVisible              bool                         `json:"scrollbar-visible,omitempty"`
	ScrollWheelMoveMultiplier     int                          `json:"scroll-wheel-move-multiplier,omitempty"`
	SendEncoding                  string                       `json:"send-encoding,omitempty"`
	ShiftInsertPaste              bool                         `json:"shift-insert-paste,omitempty"`
	UserCss                       string                       `json:"user-css,omitempty"`
}
