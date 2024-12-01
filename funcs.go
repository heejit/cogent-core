package main

import (
	"fmt"
	"database/sql"
	"os"
	"path/filepath"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
)

func ToAnyFromInterface[T any](pAny any, pDefault T) T {
	var vT T
	var vOk bool

	vT, vOk = pAny.(T)
	if vOk == false {
		return pDefault
	}
	return vT
}

func GetExecutableFileDirName() string {
	var vExePath string
	var vErr error

	vExePath, vErr = os.Executable()
	if vErr != nil {
		panic(vErr)
	}

	vExePath, vErr = filepath.EvalSymlinks(vExePath)
	if vErr != nil {
		panic(vErr)
	}

	return filepath.Dir(vExePath)

}

func ConfirmDialog(ctx core.Widget,  pMsg string, okFunc func(e events.Event)) {
	d := core.NewBody("Confirm")
	core.NewText(d).SetType(core.TextSupporting).SetText(pMsg)
	d.AddBottomBar(func(bar *core.Frame) {
		core.NewStretch(bar)
		d.AddOK(bar).SetText("Yes").OnClick(okFunc)
		d.AddCancel(bar).SetText("No").OnClick(func(e events.Event) {} )
		core.NewStretch(bar)
	})
	d.RunDialog(ctx)
}


func NextId(dbCon *sql.DB) int {
	var vSql string
	var vRows *sql.Rows
	var vErr error
	var vInt int

	vSql = " select coalesce(max(note_id), 0) + 1 from main.notes "
	vRows, vErr = dbCon.Query(vSql)
	if vErr != nil {
		fmt.Println("Error : ", vErr.Error())
		return 0
	}
	defer vRows.Close()

	vRows.Next()
	vRows.Scan(&vInt)
	return vInt
}
