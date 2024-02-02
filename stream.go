// Copyright 2024 The Contaiman Author
// This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0
// that can be found in the LICENSE file and https://mozilla.org/MPL/2.0/.

package main

import "io"

type MultiOutput struct {
	Outputs []io.Writer
}

func (m *MultiOutput) Write(b []byte) (n int, err error) {
	for _, o := range m.Outputs {
		n, err = o.Write(b)
		if err != nil {
			panic(err)
		}
	}

	return n, nil
}
