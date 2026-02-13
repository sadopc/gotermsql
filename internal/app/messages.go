package app

// Message types have moved to github.com/sadopc/gotermsql/internal/msg.
// This file re-exports them for convenience within the app package.

import appmsg "github.com/sadopc/gotermsql/internal/msg"

// Re-export types used within app package.
type (
	Pane               = appmsg.Pane
	KeyMode            = appmsg.KeyMode
	VimState           = appmsg.VimState
	ConnectMsg         = appmsg.ConnectMsg
	ConnectErrMsg      = appmsg.ConnectErrMsg
	DisconnectMsg      = appmsg.DisconnectMsg
	SchemaLoadedMsg    = appmsg.SchemaLoadedMsg
	SchemaErrMsg       = appmsg.SchemaErrMsg
	ExecuteQueryMsg    = appmsg.ExecuteQueryMsg
	QueryStartedMsg    = appmsg.QueryStartedMsg
	QueryResultMsg     = appmsg.QueryResultMsg
	QueryErrMsg       = appmsg.QueryErrMsg
	QueryStreamingMsg = appmsg.QueryStreamingMsg
	NewTabMsg         = appmsg.NewTabMsg
	CloseTabMsg        = appmsg.CloseTabMsg
	SwitchTabMsg       = appmsg.SwitchTabMsg
	StatusMsg          = appmsg.StatusMsg
	ToggleKeyModeMsg   = appmsg.ToggleKeyModeMsg
	InsertTextMsg      = appmsg.InsertTextMsg
	ExportCompleteMsg  = appmsg.ExportCompleteMsg
	ExportErrMsg       = appmsg.ExportErrMsg
)

// Re-export constants.
const (
	PaneSidebar     = appmsg.PaneSidebar
	PaneEditor      = appmsg.PaneEditor
	PaneResults     = appmsg.PaneResults
	KeyModeStandard = appmsg.KeyModeStandard
	KeyModeVim      = appmsg.KeyModeVim
	VimNormal       = appmsg.VimNormal
	VimInsert       = appmsg.VimInsert
	VimVisual       = appmsg.VimVisual
)

// Re-export functions.
var ParseKeyMode = appmsg.ParseKeyMode
