// Copyright 2024 The Contaiman Author
// This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0
// that can be found in the LICENSE file and https://mozilla.org/MPL/2.0/.

package main

import (
	"errors"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/containers/podman/v2/pkg/domain/entities"
	createconfig "github.com/containers/podman/v2/pkg/spec"
	"github.com/docker/go-connections/nat"
	. "github.com/jellytea/formui"
	"github.com/opencontainers/runtime-spec/specs-go"
	"io"
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
	dialog.NewCustom("Error", "Close", widget.NewLabel(err.Error()), w).Show()
}

func runCommand(w Window, cmd string, onExecuted func()) {
	commandEntry := widget.NewMultiLineEntry()
	commandEntry.SetText(cmd)

	ResizeDialog(600, 0, w.NewDialogWithConfirm("Run Command", "Execute", "Cancel", func() {
		log := ""

		outputEntry := widget.NewMultiLineEntry()
		outputEntry.OnChanged = func(s string) { outputEntry.SetText(log) }

		d, done := w.NewDialogWithWait("Output", outputEntry)
		defer done()

		ResizeDialog(600, 400, d).Show()

		cmd := exec.Command("bash", "-c", commandEntry.Text)
		cmd.Stdout = io.MultiWriter(os.Stdout, &TransparentWriter{OnWrite: func(b []byte) (int, error) {
			log += string(b)
			outputEntry.SetText(log)
			return 0, nil
		}})
		cmd.Stdin = os.Stdin

		err := cmd.Start()
		if err != nil {
			w.ShowError(err)
			return
		}

		err = cmd.Wait()
		if err != nil {
			outputEntry.Text += fmt.Sprint("[ Process ", err.Error(), " ]")
			outputEntry.Refresh()
			w.ShowError(err)
			return
		}

		onExecuted()
	}, commandEntry)).Show()
}

func showAbout(w fyne.Window) {
	dialog.NewCustom("Copyright \u00a9 2024 The Contaiman Author", "Close", widget.NewForm(
		widget.NewFormItem("Author", widget.NewHyperlink("Jelly Tea", &url.URL{Scheme: "https", Host: "github.com", Path: "/jellytea"})),
		widget.NewFormItem("License", widget.NewHyperlink("Mozilla Public License version 2", &url.URL{Scheme: "https", Host: "mozilla.org", Path: "/MPL/2.0/"})),
	), w).Show()
}

func showConnectionHelp(w Window, socketEntry *widget.Entry) {
	w.NewDialog("Help", container.NewVBox(
		widget.NewButton("Start userspace daemon", func() {
			runCommand(w, "systemctl start --user podman", func() {})
		}),
		widget.NewButton("Start system daemon", func() {
			runCommand(w, "systemctl start podman", func() {})
		}),
		widget.NewButton("Fill with userspace socket", func() {
			if XDG_RUNTIME_DIR, ok := os.LookupEnv("XDG_RUNTIME_DIR"); ok {
				socketEntry.SetText("unix://" + XDG_RUNTIME_DIR + "/podman/podman.sock")
			} else {
				showError(w, errors.New("environment variable $XDG_RUNTIME_DIR is undefined"))
			}
		}),
		widget.NewButton("Fill with system socket", func() {
			socketEntry.SetText("unix:///run/podman/podman.sock")
		}),
	)).Show()
}

func newHostDialog() {
	w := Window{Window: a.NewWindow("Connect to Podman instance - Contaiman")}

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
		Window: Window{w},
		Client: client,
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
	Window
	*Client
}

func (s *_Session) pullImage() {
	var (
		registryEntry = widget.NewSelectEntry(registries)
		imageEntry    = widget.NewEntry()

		archEntry    = widget.NewEntry()
		osEntry      = widget.NewEntry()
		variantEntry = widget.NewEntry()

		usernameEntry = widget.NewEntry()
		passwordEntry = widget.NewPasswordEntry()
		authFileEntry = widget.NewEntry()
		certDirEntry  = widget.NewEntry()
	)

	ResizeDialog(400, 0, s.NewFormDialogWithConfirm("Pull Image", "Pull", "Cancel",
		func() {
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
			d := s.ShowOperationDialog("Pulling image... check standard output for more detail.")
			images, err := s.PullImage(imageEntry.Text, options)
			if err != nil {
				d.Hide()
				s.ShowError(err)
				return
			}
			d.Hide()
			ResizeDialog(600, 400, s.NewDialog("Pulled Images", NewListView(images))).Show()
		},
		widget.NewFormItem("Registry", registryEntry),
		widget.NewFormItem("Image", imageEntry),
		widget.NewFormItem("Auth", widget.NewButton("Settings", func() {
			ResizeDialog(600, 0, s.NewFormDialog("Authentication",
				widget.NewFormItem("Username", usernameEntry),
				widget.NewFormItem("Password", passwordEntry),
				widget.NewFormItem("Auth File", s.NewFileOpenEntry(authFileEntry)),
				widget.NewFormItem("Cert Dir", s.NewDirOpenEntry(certDirEntry))),
			).Show()
		})),
		widget.NewFormItem("Platform", widget.NewButton("Settings", func() {
			ResizeDialog(400, 0, s.NewFormDialog("Platform",
				widget.NewFormItem("Architecture", archEntry),
				widget.NewFormItem("OS", osEntry),
				widget.NewFormItem("Variant", variantEntry),
			)).Show()
		})),
	)).Show()
}

func (s *_Session) createPod() {
	var (
		nameEntry = widget.NewEntry()
	)

	ResizeDialog(400, 0, s.NewFormDialogWithConfirm("Create Pod", "Create", "Cancel", func() {
	},
		widget.NewFormItem("Name", nameEntry),
		widget.NewFormItem("Name", nameEntry),
	)).Show()
}

func (s *_Session) createContainer() {
	s.selectImage(func(imageId string) {
		var (
			mounts []any

			nameEntry = widget.NewEntry()
			podEntry  = widget.NewSelectEntry([]string{"infra"})

			securityLabelEntry = widget.NewSelectEntry([]string{"disable"})

			detachCheck      = widget.NewCheck("-d", func(b bool) {})
			interactiveCheck = widget.NewCheck("-i", func(b bool) {})
			permanentCheck   = widget.NewCheck("-t", func(b bool) {})
			purgeCheck       = widget.NewCheck("--rm", func(b bool) {})
			privilegedCheck  = widget.NewCheck("--privileged", func(b bool) {})

			appTabs *container.AppTabs

			networkTab, getNetworkConfig = s.newNetworkTab()
		)

		var mountList *ListEdit
		mountList = NewListEdit(mounts, func(e any) string {
			return e.(*specs.Mount).Source + " => " + e.(*specs.Mount).Destination
		}, []string{
			"Add Bind Mount",
			"Add Volume",
			"Add tmpfs",
			"Add devpts",
		},
			func() {
				hostPathEntry := widget.NewEntry()
				containerPathEntry := widget.NewEntry()

				ResizeDialog(600, 0, s.NewFormDialogWithConfirm("Bind Mount", "Add", "Cancel", func() {
					mountList.Append(&specs.Mount{
						Destination: containerPathEntry.Text,
						Type:        "bind",
						Source:      hostPathEntry.Text,
						Options:     nil,
					})
				},
					widget.NewFormItem("Host Path", s.NewDirOpenEntry(hostPathEntry)),
					widget.NewFormItem("Container Path", containerPathEntry),
				)).Show()
			},
			func() {
				volumeEntry := widget.NewSelectEntry([]string{}) // TODO
				containerPathEntry := widget.NewEntry()

				ResizeDialog(600, 0, s.NewFormDialogWithConfirm("Mount Volume", "Add", "Cancel", func() {
					mountList.Append(&specs.Mount{
						Destination: containerPathEntry.Text,
						Type:        "volume",
						Source:      volumeEntry.Text,
						Options:     nil,
					})
				},
					widget.NewFormItem("Volume", volumeEntry),
					widget.NewFormItem("Container Path", containerPathEntry),
				)).Show()
			},
			func() {
				destinationEntry := widget.NewEntry()

				ResizeDialog(600, 0, s.NewFormDialogWithConfirm("Mount tmpfs", "Add", "Cancel", func() {
					mountList.Append(&specs.Mount{
						Destination: destinationEntry.Text,
						Type:        "tmpfs",
						Source:      "",
						Options:     nil,
					})
				},
				)).Show()
			},
			func() {
				destinationEntry := widget.NewEntry()

				ResizeDialog(600, 0, s.NewFormDialogWithConfirm("Mount devpts", "Add", "Cancel", func() {
					mountList.Append(&specs.Mount{
						Destination: destinationEntry.Text,
						Type:        "devpts",
						Source:      "",
						Options:     nil,
					})
				},
				)).Show()
			},
		)

		podEntry.OnChanged = func(s string) {
			if s == "" {
				appTabs.EnableIndex(3)
			} else {
				appTabs.DisableIndex(3)
			}
		}

		appTabs = container.NewAppTabs(
			container.NewTabItem("General", widget.NewForm(
				widget.NewFormItem("Image", widget.NewLabel(imageId)),
				widget.NewFormItem("Name", nameEntry),
				widget.NewFormItem("Pod", podEntry),
			)),
			container.NewTabItem("Flags", widget.NewForm(
				widget.NewFormItem("Detach", detachCheck),
				widget.NewFormItem("Interactive", interactiveCheck),
				widget.NewFormItem("Permanent", permanentCheck),
				widget.NewFormItem("Purge after stopped", purgeCheck),
				widget.NewFormItem("Privileged", privilegedCheck),
			)),
			container.NewTabItem("Mount", mountList.Widget),
			networkTab,
			container.NewTabItem("SELinux", widget.NewForm(
				widget.NewFormItem("Label", securityLabelEntry),
			)),
		)

		s.NewDialogWithConfirm("Create Container", "Create", "Cancel",
			func() {
				var mounts []specs.Mount

				for _, m := range mountList.Elements {
					mounts = append(mounts, m.(specs.Mount))
				}

				_, err := s.CreateContainer(nameEntry.Text, &createconfig.CreateConfig{
					Annotations:       nil,
					Args:              nil,
					CidFile:           "",
					ConmonPidFile:     "",
					Command:           nil,
					UserCommand:       nil,
					Detach:            false,
					Devices:           nil,
					Entrypoint:        nil,
					Env:               nil,
					HealthCheck:       nil,
					Init:              false,
					InitPath:          "",
					Image:             "",
					ImageID:           imageId,
					RawImageName:      "",
					BuiltinImgVolumes: nil,
					ImageVolumeType:   "",
					Interactive:       false,
					Labels:            nil,
					LogDriver:         "",
					LogDriverOpt:      nil,
					Name:              "",
					PodmanPath:        "",
					Pod:               "",
					Quiet:             false,
					Resources:         createconfig.CreateResourceConfig{},
					RestartPolicy:     "",
					Rm:                false,
					Rmi:               false,
					StopSignal:        0,
					StopTimeout:       0,
					Systemd:           false,
					Tmpfs:             nil,
					Tty:               false,
					Mounts:            mounts,
					MountsFlag:        nil,
					NamedVolumes:      nil,
					Volumes:           nil,
					VolumesFrom:       nil,
					WorkDir:           "",
					Rootfs:            "",
					Security:          createconfig.SecurityConfig{},
					Syslog:            false,
					Pid:               createconfig.PidConfig{},
					Ipc:               createconfig.IpcConfig{},
					Cgroup:            createconfig.CgroupConfig{},
					User:              createconfig.UserConfig{},
					Uts:               createconfig.UtsConfig{},
					Network:           getNetworkConfig(),
				})
				if err != nil {
					s.ShowError(err)
					return
				}
				dialog.NewInformation("Container ID", imageId, s.Window)
			}, appTabs,
		).Show()

	})
}

func (s *_Session) newNetworkTab() (*container.TabItem, func() createconfig.NetworkConfig) {
	var (
		exportedPorts []any
		dnsServers    []any

		networkEntry = widget.NewSelectEntry(nil)

		macAddrEntry = widget.NewEntry()
		ipv4Entry    = widget.NewEntry()
		ipv6Entry    = widget.NewEntry()

		dnsOptsEntry  = widget.NewMultiLineEntry()
		dnsServerList *ListEdit

		portList *ListEdit
	)

	portList = NewListEdit(exportedPorts, func(e any) string {
		return e.(nat.Port).Port()
	}, []string{"Add"}, func() {
		var (
			protoEntry = widget.NewSelectEntry([]string{"tcp", "udp"})
			portEntry  = widget.NewEntry()
		)

		ResizeDialog(400, 400, s.NewFormDialogWithConfirm("Add Port", "Add", "Cancel", func() {
			port, err := nat.NewPort(protoEntry.Text, portEntry.Text)
			if err != nil {
				s.ShowError(err)
				return
			}
			portList.Append(port)
		},
			widget.NewFormItem("Protocol", protoEntry),
			widget.NewFormItem("Port", portEntry),
		)).Show()
	})
	dnsServerList = NewListEdit(dnsServers, func(e any) string {
		return e.(string)
	}, []string{"Add DNS Server"}, func() {
		dnsServerEntry := widget.NewEntry()
		ResizeDialog(400, 0, s.NewFormDialogWithConfirm("Add DNS Server", "Add", "Cancel", func() {
			dnsServerList.Append(dnsServerEntry.Text)
		},
			widget.NewFormItem("Server", dnsServerEntry),
		)).Show()
	})

	return container.NewTabItem("Network", widget.NewForm(
			widget.NewFormItem("Network", networkEntry),
			widget.NewFormItem("Address", widget.NewButton("Settings", func() {
				ResizeDialog(400, 0, s.NewFormDialog("Address",
					widget.NewFormItem("MAC Address", macAddrEntry),
					widget.NewFormItem("IPv4 Address", ipv4Entry),
					widget.NewFormItem("IPv6 Address", ipv6Entry),
				)).Show()
			})),
			widget.NewFormItem("DNS", widget.NewButton("Settings", func() {
				ResizeDialog(400, 0, s.NewFormDialog("DNS",
					widget.NewFormItem("DNS Servers", dnsServerList.Widget),
					widget.NewFormItem("resolve.conf", dnsOptsEntry),
				)).Show()
			})),
			widget.NewFormItem("Ports", widget.NewButton("Settings", func() {
				ResizeDialog(400, 600, s.NewDialog("Ports", portList.Widget)).Show()
			})),
		)), func() createconfig.NetworkConfig {
			ports := map[nat.Port]struct{}{}
			for _, port := range AnyArrayCast[nat.Port](exportedPorts) {
				ports[port] = struct{}{}
			}

			return createconfig.NetworkConfig{
				DNSOpt:       nil,
				DNSSearch:    nil,
				DNSServers:   AnyArrayCast[string](dnsServers),
				ExposedPorts: ports,
				HTTPProxy:    false,
				IP6Address:   "",
				IPAddress:    "",
				LinkLocalIP:  nil,
				MacAddress:   macAddrEntry.Text,
				NetMode:      "",
				Network:      "",
				NetworkAlias: nil,
				PortBindings: nil,
				Publish:      nil,
				PublishAll:   false,
			}
		}
}

func (s *_Session) selectContainer() {
	containers, err := s.QueryContainers()
	if err != nil {
		s.ShowError(err)
		return
	}
	var list []string
	for _, c := range containers {
		list = append(list, c.ID)
	}

	s.NewDialogWithConfirm("Select Container", "Select", "Cancel", func() {
		// TODO
	}, NewListSelect(func(i int, v string) {
		// TODO
	}, list)).Show()
}

func (s *_Session) selectImage(callback func(id string)) {
	images, err := s.QueryImages()
	if err != nil {
		s.ShowError(err)
		return
	}
	var labels []string
	for _, image := range images {
		labels = append(labels, image.ID)
	}

	var id = -1

	list := NewListSelect(func(i int, v string) {
		id = i
	}, labels)

	ResizeDialog(600, 400, s.NewDialogWithConfirm("Select Image", "Select", "Cancel", func() {
		if id < 0 {
			dialog.NewInformation("Select Image", "No image selected.", s.Window).Show()
			return
		}
		callback(labels[id])
	}, list)).Show()
}
