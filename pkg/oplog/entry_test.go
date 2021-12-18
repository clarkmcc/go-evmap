/*
Copyright (C) 2020 Print Tracker, LLC - All Rights Reserved

Unauthorized copying of this file, via any medium is strictly prohibited
as this source code is proprietary and confidential. Dissemination of this
information or reproduction of this material is strictly forbidden unless
prior written permission is obtained from Print Tracker, LLC.
*/

package oplog

import (
	"fmt"
	"testing"
)

func Test(t *testing.T) {
	v := "bar"
	e := Insert("foo", &v)
	fmt.Println(*e.v)
}
