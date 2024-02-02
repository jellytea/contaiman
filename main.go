// Copyright 2024 The Contaiman Author
// This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0
// that can be found in the LICENSE file and https://mozilla.org/MPL/2.0/.

package main

import (
	"errors"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/containers/podman/v2/pkg/domain/entities"
	"net/url"
	"os"
	"os/exec"
)

var registries = []string{
	"registry.redhat.io",
	"docker.io",
}

var a = app.New()

func main() {
	newHostDialog()
	a.Run()
}

func showError(w fyne.Window, err error) {
	if err == nil {
		return
	}
	dialog.NewCustom("Error", "Close", widget.NewLabel(err.Error()), w).Show()
}

func runCommand(w fyne.Window, cmd string, onExecuted func()) {
	commandEntry := widget.NewMultiLineEntry()
	commandEntry.SetText(cmd)

	d := dialog.NewCustomConfirm("Run Command", "Execute", "Cancel", commandEntry, func(b bool) {
		if b {
			err := exec.Command("bash", "-c", commandEntry.Text).Run()
			if err != nil {
				showError(w, err)
				return
			}
			onExecuted()
		}
	}, w)
	d.Resize(fyne.NewSize(600, 0))
	d.Show()
}

func showAbout(w fyne.Window) {
	dialog.NewCustom("Copyright \u00a9 2024 The Contaiman Author", "Close", widget.NewForm(
		widget.NewFormItem("Author", widget.NewHyperlink("Jelly Tea", &url.URL{Scheme: "https", Host: "github.com", Path: "/jellytea"})),
		widget.NewFormItem("License", widget.NewHyperlink("Mozilla Public License version 2", &url.URL{Scheme: "https", Host: "mozilla.org", Path: "/MPL/2.0/"})),
	), w).Show()
}

func showConnectionHelp(w fyne.Window, socketEntry *widget.Entry) {
	var dispose func()
	d := dialog.NewCustom("Help with your Podman", "Close", container.NewVBox(
		widget.NewButton("Start userspace daemon", func() {
			runCommand(w, "systemctl start --user podman", func() { dispose() })
		}),
		widget.NewButton("Start system daemon", func() {
			runCommand(w, "systemctl start podman", func() { dispose() })
		}),
		widget.NewButton("Fill with userspace socket", func() {
			if XDG_RUNTIME_DIR, ok := os.LookupEnv("XDG_RUNTIME_DIR"); ok {
				socketEntry.SetText("unix://" + XDG_RUNTIME_DIR + "/podman/podman.sock")
				dispose()
			} else {
				showError(w, errors.New("environment variable $XDG_RUNTIME_DIR is undefined"))
			}
		}),
		widget.NewButton("Fill with system socket", func() {
			socketEntry.SetText("unix:///run/podman/podman.sock")
			dispose()
		}),
	), w)
	dispose = func() { d.Hide() }
	d.Show()
}

func newHostDialog() {
	w := a.NewWindow("Connect to Podman instance - Contaiman")

	socketEntry := widget.NewEntry()

	w.SetContent(container.NewCenter(container.NewVBox(
		widget.NewLabel("Connect to the Podman instance, enter the URI below:"),
		socketEntry,
		widget.NewButton("Connect", func() {
			client, err := NewClient(socketEntry.Text)
			if err != nil {
				showError(w, err)
				return
			}
			newHost(w, &client)
		}),
		widget.NewButton("Help", func() { showConnectionHelp(w, socketEntry) }),
		widget.NewButton("About", func() { showAbout(w) }),
	)))

	w.Resize(fyne.NewSize(800, 600))
	w.Show()
}

func newHost(w fyne.Window, client *Client) {
	s := _Session{
		w: w,
		c: client,
	}

	w.SetTitle(client.Host + " - Contaiman")

	w.SetContent(container.NewBorder(nil, nil, nil, nil, nil))

	w.SetMainMenu(
		fyne.NewMainMenu(
			fyne.NewMenu("Client",
				fyne.NewMenuItem("New Connection", func() { newHostDialog() }),
				fyne.NewMenuItem("Quit", func() { w.Close() }),
			),
			fyne.NewMenu("Containers",
				fyne.NewMenuItem("Create", func() { s.createContainer() }),
			),
			fyne.NewMenu("Images",
				fyne.NewMenuItem("Pull", func() { s.pullImage() }),
			),
			fyne.NewMenu("Pods",
				fyne.NewMenuItem("Create", func() { s.createPod() }),
			),
			fyne.NewMenu("About",
				fyne.NewMenuItem("About Contaiman", func() { showAbout(w) }),
			),
		),
	)
}

type _Session struct {
	w fyne.Window
	c *Client
}

func (s *_Session) pullImage() {
	registryEntry := widget.NewEntry()
	imageEntry := widget.NewEntry()

	archEntry := widget.NewEntry()
	osEntry := widget.NewEntry()
	variantEntry := widget.NewEntry()

	usernameEntry := widget.NewEntry()
	passwordEntry := widget.NewPasswordEntry()
	authFileEntry := widget.NewEntry()
	certDirEntry := widget.NewEntry()

	d := dialog.NewForm("Pull Image", "Pull", "Cancel", []*widget.FormItem{
		widget.NewFormItem("Registry", container.NewVBox(
			registryEntry,
			widget.NewButton("...", func() {
				s.ShowListSelectDialog("Registries", "Select", "Close", registries, func(i int) {
					registryEntry.SetText(registries[i])
				})
			}),
		)),
		widget.NewFormItem("Image", imageEntry),
		widget.NewFormItem("Auth", widget.NewButton("Settings", func() {
			d := dialog.NewCustom("Authentication", "Close", widget.NewForm(
				widget.NewFormItem("Username", usernameEntry),
				widget.NewFormItem("Password", passwordEntry),
				widget.NewFormItem("Auth File", s.NewFileEntry(authFileEntry)),
				widget.NewFormItem("Cert Dir", s.NewDirEntry(certDirEntry)),
			), s.w)
			d.Resize(fyne.NewSize(600, 0))
			d.Show()
		})),
		widget.NewFormItem("Platform", widget.NewButton("Settings", func() {
			d := dialog.NewCustom("Platform", "Close", widget.NewForm(
				widget.NewFormItem("Architecture", archEntry),
				widget.NewFormItem("OS", osEntry),
				widget.NewFormItem("Variant", variantEntry),
			), s.w)
			d.Resize(fyne.NewSize(400, 0))
			d.Show()
		})),
	}, func(b bool) {
		if b {
			options := entities.ImagePullOptions{
				AllTags:         false,
				Authfile:        authFileEntry.Text,
				CertDir:         certDirEntry.Text,
				Username:        usernameEntry.Text,
				Password:        passwordEntry.Text,
				OverrideArch:    archEntry.Text,
				OverrideOS:      osEntry.Text,
				OverrideVariant: variantEntry.Text,
				Quiet:           false,
				SignaturePolicy: "",
				SkipTLSVerify:   0,
				PullPolicy:      0,
			}
			images, err := s.c.PullImage(imageEntry.Text, options)
			if err != nil {
				s.ShowError(err)
				return
			}
			s.ShowListSelectDialog("Pulled images", "Okay", "Close", images, func(i int) {
			})
		}
	}, s.w)

	d.Resize(fyne.NewSize(400, 0))
	d.Show()
}

func (s *_Session) createPod() {}

func (s *_Session) createContainer() {
	s.selectImage()
}

func (s *_Session) selectContainer() {
	containers, err := s.c.QueryContainers()
	if err != nil {
		showError(s.w, err)
		return
	}

	list := widget.NewList(func() int {
		return len(containers)
	}, func() fyne.CanvasObject {
		return widget.NewLabel("")
	}, func(id widget.ListItemID, object fyne.CanvasObject) {
	})

	dialog.NewCustomConfirm("Select Container", "Select", "Cancel", list, func(b bool) {

	}, s.w).Show()
}

func (s *_Session) selectImage() {
	images, err := s.c.QueryImages()
	if err != nil {
		s.ShowError(err)
		return
	}
	var labels []string
	for _, image := range images {
		labels = append(labels, image.ID)
	}

	list := NewList(labels, func(i int) {
	})

	d := dialog.NewCustomConfirm("Select Image", "Select", "Cancel", container.NewVBox(
		list,
	), func(b bool) {
		if b {
			//s.c.CreateContainer()
		}
	}, s.w)
	d.Resize(fyne.NewSize(600, 400))
	d.Show()
}
