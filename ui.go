// Copyright 2024 The Contaiman Author
// This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0
// that can be found in the LICENSE file and https://mozilla.org/MPL/2.0/.

package main

import (
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func (s *_Session) ShowOperationDialog(message string) dialog.Dialog {
	d := dialog.NewCustomWithoutButtons("Operating", widget.NewLabel(message), s.Window)
	d.Show()
	return d
}
