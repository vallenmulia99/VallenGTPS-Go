package dialog

import (
	"fmt"
	"strings"
)

type DialogBuilder struct {
	parts []string
}

func New() *DialogBuilder {
	return &DialogBuilder{
		parts: make([]string, 0),
	}
}

func (d *DialogBuilder) SetDefaultColor(code string) *DialogBuilder {
	d.parts = append(d.parts, fmt.Sprintf("set_default_color|%s", code))
	return d
}

func (d *DialogBuilder) AddLabel(size, text string) *DialogBuilder {
	d.parts = append(d.parts, fmt.Sprintf("add_label|%s|%s|left", size, text))
	return d
}

func (d *DialogBuilder) AddLabelWithIcon(size, text string, iconID int) *DialogBuilder {
	d.parts = append(d.parts, fmt.Sprintf("add_label_with_icon|%s|%s|left|%d|", size, text, iconID))
	return d
}

func (d *DialogBuilder) AddTextbox(text string) *DialogBuilder {
	d.parts = append(d.parts, fmt.Sprintf("add_textbox|%s|left|", text))
	return d
}

func (d *DialogBuilder) AddSmallText(text string) *DialogBuilder {
	d.parts = append(d.parts, fmt.Sprintf("add_smalltext|%s|left|", text))
	return d
}

func (d *DialogBuilder) AddTextInput(id, label, defaultVal string, maxLen int) *DialogBuilder {
	d.parts = append(d.parts, fmt.Sprintf("add_text_input|%s|%s|%s|%d|", id, label, defaultVal, maxLen))
	return d
}

func (d *DialogBuilder) AddTextInputInt(id, label string, defaultVal int, maxLen int) *DialogBuilder {
	return d.AddTextInput(id, label, fmt.Sprintf("%d", defaultVal), maxLen)
}

func (d *DialogBuilder) AddSpacer(size string) *DialogBuilder {
	d.parts = append(d.parts, fmt.Sprintf("add_spacer|%s|", size))
	return d
}

func (d *DialogBuilder) AddCustomMargin(x, y int) *DialogBuilder {
	d.parts = append(d.parts, fmt.Sprintf("add_custom_margin|x:%d;y:%d|", x, y))
	return d
}

func (d *DialogBuilder) SetCustomSpacing(x, y int) *DialogBuilder {
	d.parts = append(d.parts, fmt.Sprintf("set_custom_spacing|x:%d;y:%d|", x, y))
	return d
}

func (d *DialogBuilder) AddButton(btnID, btnName string) *DialogBuilder {
	d.parts = append(d.parts, fmt.Sprintf("add_button|%s|%s|noflags|0|0|", btnID, btnName))
	return d
}

func (d *DialogBuilder) AddCustomButton(btnID, imagePath string) *DialogBuilder {
	d.parts = append(d.parts, fmt.Sprintf("add_custom_button|%s|%s|", btnID, imagePath))
	return d
}

func (d *DialogBuilder) AddCheckbox(id, label string, checked bool) *DialogBuilder {
	flag := "0"
	if checked {
		flag = "1"
	}
	d.parts = append(d.parts, fmt.Sprintf("add_checkbox|%s|%s|%s", id, label, flag))
	return d
}

func (d *DialogBuilder) AddRadioButton(groupID, radioID, label string, selected bool) *DialogBuilder {
	flag := "0"
	if selected {
		flag = "1"
	}
	d.parts = append(d.parts, fmt.Sprintf("add_radio_button|%s|%s|%s|%s|", groupID, radioID, label, flag))
	return d
}

func (d *DialogBuilder) EmbedData(id string, val interface{}) *DialogBuilder {
	d.parts = append(d.parts, fmt.Sprintf("embed_data|%s|%v", id, val))
	return d
}

func (d *DialogBuilder) AddQuickExit() *DialogBuilder {
	d.parts = append(d.parts, "add_quick_exit|")
	return d
}

func (d *DialogBuilder) AddPopupName(name string) *DialogBuilder {
	d.parts = append(d.parts, fmt.Sprintf("add_popup_name|%s", name))
	return d
}

func (d *DialogBuilder) AddPlayerInfo(label, progressBarName string, progress, total int) *DialogBuilder {
	d.parts = append(d.parts, fmt.Sprintf("add_player_info|%s|%s|%d|%d|", label, progressBarName, progress, total))
	return d
}

func (d *DialogBuilder) AddPlayerPicker(id, label string) *DialogBuilder {
	d.parts = append(d.parts, fmt.Sprintf("add_player_picker|%s|%s|", id, label))
	return d
}

func (d *DialogBuilder) EndDialog(dialogName, cancelBtn, okBtn string) string {
	d.parts = append(d.parts, fmt.Sprintf("end_dialog|%s|%s|%s|", dialogName, cancelBtn, okBtn))
	return d.build()
}

func (d *DialogBuilder) Build() string {
	return d.build()
}

func (d *DialogBuilder) build() string {
	return strings.Join(d.parts, "\n") + "\n"
}
// ─────────────────────────────────────────────
// Instant Dialog Factories (Lazy / Clean Style)
// ─────────────────────────────────────────────

// Alert membuat dialog pop-up pemberitahuan sederhana dengan 1 tombol konfirmasi.
func Alert(title string, iconID int, text string, okBtn string) string {
	if okBtn == "" {
		okBtn = "OK"
	}
	d := New().SetDefaultColor("`o")
	if iconID > 0 {
		d.AddLabelWithIcon("big", fmt.Sprintf("`w%s``", title), iconID)
	} else {
		d.AddLabel("big", fmt.Sprintf("`w%s``", title))
	}
	d.AddSpacer("small")
	d.AddTextbox(text)
	d.AddSpacer("small")
	d.AddQuickExit()
	return d.EndDialog("alert_popup", "", okBtn)
}

// Confirm membuat dialog konfirmasi dengan tombol Batal dan OK.
func Confirm(dialogName, title string, iconID int, text string, cancelBtn, okBtn string) string {
	if cancelBtn == "" {
		cancelBtn = "Cancel"
	}
	if okBtn == "" {
		okBtn = "OK"
	}
	d := New().SetDefaultColor("`o")
	if iconID > 0 {
		d.AddLabelWithIcon("big", fmt.Sprintf("`w%s``", title), iconID)
	} else {
		d.AddLabel("big", fmt.Sprintf("`w%s``", title))
	}
	d.AddSpacer("small")
	d.AddTextbox(text)
	d.AddSpacer("small")
	d.AddQuickExit()
	return d.EndDialog(dialogName, cancelBtn, okBtn)
}

// Input membuat dialog dengan kotak input teks, tombol Batal dan OK.
func Input(dialogName, title string, iconID int, text string, inputID, label, defaultVal string, maxLen int, cancelBtn, okBtn string) string {
	if cancelBtn == "" {
		cancelBtn = "Cancel"
	}
	if okBtn == "" {
		okBtn = "OK"
	}
	d := New().SetDefaultColor("`o")
	if iconID > 0 {
		d.AddLabelWithIcon("big", fmt.Sprintf("`w%s``", title), iconID)
	} else {
		d.AddLabel("big", fmt.Sprintf("`w%s``", title))
	}
	d.AddSpacer("small")
	if text != "" {
		d.AddTextbox(text)
	}
	d.AddTextInput(inputID, label, defaultVal, maxLen)
	d.AddSpacer("small")
	d.AddQuickExit()
	return d.EndDialog(dialogName, cancelBtn, okBtn)
}
