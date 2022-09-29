/*
 *
 *  * Copyright 2021 KubeClipper Authors.
 *  *
 *  * Licensed under the Apache License, Version 2.0 (the "License");
 *  * you may not use this file except in compliance with the License.
 *  * You may obtain a copy of the License at
 *  *
 *  *     http://www.apache.org/licenses/LICENSE-2.0
 *  *
 *  * Unless required by applicable law or agreed to in writing, software
 *  * distributed under the License is distributed on an "AS IS" BASIS,
 *  * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  * See the License for the specific language governing permissions and
 *  * limitations under the License.
 *
 */

package fileutil

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBackup(t *testing.T) {
	expected := `There lies the port; the vessel puffs her sail:
There gloom the dark, broad seas. My mariners,
Souls that have toil'd, and wrought, and thought with me—
That ever with a frolic welcome took
The thunder and the sunshine, and opposed
Free hearts, free foreheads—you and I are old;
Old age hath yet his honour and his toil;
Death closes all: but something ere the end,
Some work of noble note, may yet be done,
Not unbecoming men that strove with Gods.
The lights begin to twinkle from the rocks:
The long day wanes: the slow moon climbs: the deep
Moans round with many voices. Come, my friends,
'T is not too late to seek a newer world.
Push off, and sitting well in order smite
The sounding furrows; for my purpose holds
To sail beyond the sunset, and the baths
Of all the western stars, until I die.
It may be that the gulfs will wash us down:
It may be we shall touch the Happy Isles,
And see the great Achilles, whom we knew.
Tho' much is taken, much abides; and tho'
We are not now that strength which in old days
Moved earth and heaven, that which we are, we are;
One equal temper of heroic hearts,
Made weak by time and fate, but strong in will
To strive, to seek, to find, and not to yield.
`
	fileName := "test.txt"
	if err := WriteFileWithDataFunc(fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644,
		func(w io.Writer) error {
			_, err := w.Write([]byte(expected))
			return err
		}, false); err != nil {
		assert.FailNowf(t, "failed to write file", err.Error())
	}
	pwd, err := os.Getwd()
	if err != nil {
		assert.FailNowf(t, "failed get current directory", err.Error())
	}
	bakFile, err := Backup(fileName, pwd)
	if err != nil {
		assert.FailNowf(t, "failed to backup file", err.Error())
	}
	actual, err := os.ReadFile(bakFile)
	if err != nil {
		assert.FailNowf(t, "failed to read backup file", err.Error())
	}
	assert.Equal(t, expected, string(actual))
	os.Remove(fileName)
	os.Remove(bakFile)
}
