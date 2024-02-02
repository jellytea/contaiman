// Copyright 2024 The Contaiman Author
// This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0
// that can be found in the LICENSE file and https://mozilla.org/MPL/2.0/.

package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"sync"
	"time"
)

func (s *_Session) ShowError(err error) {
	showError(s.w, err)
}

func NewList(elements []string, onSelect func(i int)) *widget.List {
	list := widget.NewList(func() int {
		return len(elements)
	}, func() fyne.CanvasObject {
		return widget.NewLabel("")
	}, func(id widget.ListItemID, object fyne.CanvasObject) {
		object.(*widget.Label).SetText(elements[id])
	})
	list.OnSelected = onSelect

	return list
}

func (s *_Session) ShowListSelectDialog(title, confirm, dismiss string, elements []string, callback func(i int)) {
	list := widget.NewList(func() int {
		return len(elements)
	}, func() fyne.CanvasObject {
		return widget.NewLabel("")
	}, func(id widget.ListItemID, object fyne.CanvasObject) {
		object.(*widget.Label).SetText(elements[id])
	})
	var i = -1
	list.OnSelected = func(id widget.ListItemID) {
		i = id
	}
	d := dialog.NewCustomConfirm(title, confirm, dismiss, list, func(b bool) {
		if b {
			callback(i)
		}
	}, s.w)
	d.Resize(fyne.NewSize(400, 600))
	d.Show()
}

var bugWarner sync.Once

func bugWarn(w fyne.Window) {
	go func() {
		time.Sleep(1 * time.Second)
		dialog.NewInformation("BUG Warning", "It is known that there is a bug in File Dialog, DON'T click Cancel or the program will crash.", w).Show()
	}()
}

func (s *_Session) NewFileButton(f func(path string)) *widget.Button {
	//bugWarner.Do(func() { bugWarn(s.w) })
	return widget.NewButton("...", func() {
		dialog.NewFileOpen(func(closer fyne.URIReadCloser, err error) {
			if closer == nil {
				return
			}
			if err == nil {
				f(closer.URI().Path())
			}
		}, s.w).Show()
	})
}

func (s *_Session) NewDirButton(f func(path string)) *widget.Button {
	//bugWarner.Do(func() { bugWarn(s.w) })
	return widget.NewButton("...", func() {
		dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if uri == nil {
				return
			}
			if err == nil {
				list, err := uri.List()
				if err == nil {
					f(list[0].Path())
				}
			}
		}, s.w).Show()
	})
}

func (s *_Session) NewFileEntry(entry *widget.Entry) *fyne.Container {
	return container.NewVBox(
		entry,
		s.NewFileButton(func(path string) {
			entry.SetText(path)
		}))
}

func (s *_Session) NewDirEntry(entry *widget.Entry) *fyne.Container {
	return container.NewVBox(
		entry,
		s.NewDirButton(func(path string) {
			entry.SetText(path)
		}))
}
