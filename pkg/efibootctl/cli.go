// Copyright (c) 2022 individual contributors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// <https://www.apache.org/licenses/LICENSE-2.0>
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package efibootctl

import (
	"errors"
	"fmt"
	"os"

	"github.com/0x5a17ed/itkit/iters/sliceit"
	"github.com/0x5a17ed/itkit/itlib"
	"github.com/0x5a17ed/uefi/efi/efitypes"
	"github.com/0x5a17ed/uefi/efi/efivario"
	"github.com/0x5a17ed/uefi/efi/efivars"
	"go.uber.org/multierr"

	"github.com/0x5a17ed/efibootctl/pkg/printer"
)

type BootIndex uint16

func (i BootIndex) PrettyPrint(p *printer.Printer) {
	p.ColorPrint(fmt.Sprintf("%04X", i), printer.IntegerColor)
}

func mainE() (err error) {
	c := efivario.NewDefaultContext()
	defer multierr.AppendInvoke(&err, multierr.Close(c))

	p := printer.NewPrinter("", printer.DefaultScheme, true, true, true)

	// Report BootNext value.
	_, bootNext, err := efivars.BootNext.Get(c)
	if err != nil {
		if !errors.Is(err, efivario.ErrNotFound) {
			return err
		}
		// Ignore efivario.ErrNotFound errors.
	} else {
		p.PrintFieldValue("BootNext", BootIndex(bootNext))
	}

	// Report BootCurrent value.
	_, bootCurrent, err := efivars.BootCurrent.Get(c)
	if err != nil {
		return err
	}
	p.PrintFieldValue("BootCurrent", BootIndex(bootCurrent))

	// TODO: implement timeout value in uefi package.

	_, bootOrder, err := efivars.BootOrder.Get(c)
	if err != nil {
		return err
	}

	p.PrintFieldValue("BootOrder", sliceit.To(itlib.Map(
		sliceit.In(bootOrder), func(v uint16) BootIndex { return BootIndex(v) },
	)))

	it, err := efivars.BootIterator(c)
	if err != nil {
		return err
	}
	defer multierr.AppendInvoke(&err, multierr.Close(it))

	itlib.Apply(it.Iter(), func(be *efivars.BootEntry) {
		_, lo, err := be.Variable.Get(c)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "error: %s\n", err)
			return
		}

		var isActive string
		if lo.Attributes&efitypes.ActiveAttribute != 0 {
			isActive = "*"
		}

		p.PrintFieldValue(
			fmt.Sprintf("Boot%04X%s", be.Index, isActive),
			lo.DescriptionString(),
		)
		return
	})

	_, _ = fmt.Fprint(printer.DefaultOut, p.String())

	return nil
}

func Run(binName string, args []string) {
	if err := RunWithPrivileges(mainE); err != nil {
		fmt.Printf("error: %s\n", err.Error())
		os.Exit(1)
	}
}
