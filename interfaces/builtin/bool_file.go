// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2016 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package builtin

import (
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/snapcore/snapd/interfaces"
	"github.com/snapcore/snapd/interfaces/apparmor"
)

// boolFileInterface is the type of all the bool-file interfaces.
type boolFileInterface struct{}

// String returns the same value as Name().
func (iface *boolFileInterface) String() string {
	return iface.Name()
}

// Name returns the name of the bool-file interface.
func (iface *boolFileInterface) Name() string {
	return "bool-file"
}

var boolFileGPIOValuePattern = regexp.MustCompile(
	"^/sys/class/gpio/gpio[0-9]+/value$")
var boolFileAllowedPathPatterns = []*regexp.Regexp{
	// The brightness of standard LED class device
	regexp.MustCompile("^/sys/class/leds/[^/]+/brightness$"),
	// The value of standard exported GPIO
	boolFileGPIOValuePattern,
}

// SanitizeSlot checks and possibly modifies a slot.
// Valid "bool-file" slots must contain the attribute "path".
func (iface *boolFileInterface) SanitizeSlot(slot *interfaces.Slot) error {
	if iface.Name() != slot.Interface {
		panic(fmt.Sprintf("slot is not of interface %q", iface))
	}
	path, ok := slot.Attrs["path"].(string)
	if !ok || path == "" {
		return fmt.Errorf("bool-file must contain the path attribute")
	}
	path = filepath.Clean(path)
	for _, pattern := range boolFileAllowedPathPatterns {
		if pattern.MatchString(path) {
			return nil
		}
	}
	return fmt.Errorf("bool-file can only point at LED brightness or GPIO value")
}

// SanitizePlug checks and possibly modifies a plug.
func (iface *boolFileInterface) SanitizePlug(plug *interfaces.Plug) error {
	if iface.Name() != plug.Interface {
		panic(fmt.Sprintf("plug is not of interface %q", iface))
	}
	// NOTE: currently we don't check anything on the plug side.
	return nil
}

func (iface *boolFileInterface) AppArmorPermanentSlot(spec *apparmor.Specification, slot *interfaces.Slot) error {
	gpioSnippet := `
/sys/class/gpio/export rw,
/sys/class/gpio/unexport rw,
/sys/class/gpio/gpio[0-9]+/direction rw,
`

	if iface.isGPIO(slot) {
		spec.AddSnippet(gpioSnippet)
	}
	return nil
}

func (iface *boolFileInterface) AppArmorConnectedPlug(spec *apparmor.Specification, plug *interfaces.Plug, plugAttrs map[string]interface{}, slot *interfaces.Slot, slotAttrs map[string]interface{}) error {
	// Allow write and lock on the file designated by the path.
	// Dereference symbolic links to file path handed out to apparmor since
	// sysfs is full of symlinks and apparmor requires uses real path for
	// filtering.
	path, err := iface.dereferencedPath(slot)
	if err != nil {
		return fmt.Errorf("cannot compute plug security snippet: %v", err)
	}
	spec.AddSnippet(fmt.Sprintf("%s rwk,", path))
	return nil
}

func (iface *boolFileInterface) dereferencedPath(slot *interfaces.Slot) (string, error) {
	if path, ok := slot.Attrs["path"].(string); ok {
		path, err := evalSymlinks(path)
		if err != nil {
			return "", err
		}
		return filepath.Clean(path), nil
	}
	panic("slot is not sanitized")
}

// isGPIO checks if a given bool-file slot refers to a GPIO pin.
func (iface *boolFileInterface) isGPIO(slot *interfaces.Slot) bool {
	if path, ok := slot.Attrs["path"].(string); ok {
		path = filepath.Clean(path)
		return boolFileGPIOValuePattern.MatchString(path)
	}
	panic("slot is not sanitized")
}

// AutoConnect returns whether plug and slot should be implicitly
// auto-connected assuming they will be an unambiguous connection
// candidate and declaration-based checks allow.
//
// By default we allow what declarations allowed.
func (iface *boolFileInterface) AutoConnect(*interfaces.Plug, *interfaces.Slot) bool {
	return true
}

func (iface *boolFileInterface) ValidatePlug(plug *interfaces.Plug, attrs map[string]interface{}) error {
	return nil
}

func (iface *boolFileInterface) ValidateSlot(slot *interfaces.Slot, attrs map[string]interface{}) error {
	return nil
}

func init() {
	registerIface(&boolFileInterface{})
}
