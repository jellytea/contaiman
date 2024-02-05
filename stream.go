// Copyright 2024 The Contaiman Author
// This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0
// that can be found in the LICENSE file and https://mozilla.org/MPL/2.0/.

package main

type TransparentWriter struct {
	OnWrite func(b []byte) (int, error)
}

func (w *TransparentWriter) Write(b []byte) (int, error) {
	return w.OnWrite(b)
}
