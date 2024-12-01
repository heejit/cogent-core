package main

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	_ "modernc.org/sqlite"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/texteditor"
)


type MainForm struct {
	dbSqlite    *sql.DB
	DbFileName  string
	OldId       int

	Body        *core.Body
	SearchText  *core.TextField
	FormName    *core.TextField
	FormTag     *core.TextField
	FormContent *texteditor.Editor
	//FormContent *core.TextField

	CardGrid    *CardList
}

func main() {
	vMainForm := new(MainForm)
	vMainForm.InitForm()
	colors.SetScheme(true) // dark theme
	vMainForm.SearchText.SetFocus()
	vMainForm.Body.RunMainWindow()
}

func (this *MainForm) InitForm() {
	var vErr error

	this.Body = core.NewBody("My Notes")
	this.Body.OnFirst(events.KeyChord, func (e events.Event) {
		if e.KeyChord() == "Control+S" {
			e.SetHandled()
			this.SaveForm()
		}
	})

	vSplit := core.NewSplits(this.Body)
	vSplit.SetSplits(0.4, 0.6)
	vFrame1 := core.NewFrame(vSplit)
	vFrame1.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})
	vFrame1.ContextMenus = nil
	vFrame1.Scene.ContextMenus = nil
	this.SearchText = core.NewTextField(vFrame1)
	this.SearchText.SetPlaceholder("Type and press enter to search")
	this.SearchText.Styler(func(s *styles.Style) {
		s.Max.Zero()
		s.SetTextWrap(false)
	})
	this.SearchText.OnKeyChord(func (e events.Event) {
		if slices.Contains([]key.Chord{"ReturnEnter", "KeypadEnter"}, e.KeyChord()) == true {
			e.SetHandled()
			this.SearchNotes(this.SearchText.Text())
		}
	})

	this.CardGrid = NewCardList(vFrame1)
	this.CardGrid.SetOnClick(func(v CardData) {
		vInt := ToAnyFromInterface(v.Data, 0)
		this.ShowNote(vInt)
	})

	vFrame2 := core.NewFrame(vSplit)
	vFrame2.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})

	this.FormName = core.NewTextField(vFrame2)
	this.FormName.SetPlaceholder("Enter Name")
	this.FormName.Styler(func(s *styles.Style) {
		s.Max.Zero()
	})

	this.FormTag = core.NewTextField(vFrame2)
	this.FormTag.SetPlaceholder("Enter tag(s) comma separated")
	this.FormTag.Styler(func(s *styles.Style) {
		s.Max.Zero() // stretch
	})

	// somtime ctlr+v hang the app 
	this.FormContent = texteditor.NewEditor(vFrame2)
	this.FormContent.ContextMenus = nil
	this.FormContent.Scene.ContextMenus = nil
	this.FormContent.SetTooltip("Enter your notes here")
	this.FormContent.Styler(func (s *styles.Style) {
		s.Grow.Set(1, 1)
		s.SetTextWrap(false)
		s.SetAbilities(false, abilities.Hoverable)
	})

	vFrame3 := core.NewFrame(vFrame2)
	vFrame3.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 0)
	})
	vBtnRemove := core.NewButton(vFrame3)
	vBtnRemove.SetText("Remove")
	vBtnRemove.OnClick(this.OnRemoveClick)
	vBtnRemove.SetIcon(icons.Delete)
	vBtnRemove.Styler(func (s *styles.Style) {
		s.Background = colors.Uniform(colors.Red)
		s.Color = colors.Uniform(colors.White)
	})

	core.NewStretch(vFrame3)
	vBtnSave := core.NewButton(vFrame3)
	vBtnSave.SetText("Save")
	vBtnSave.OnClick(func(e events.Event) {
		this.SaveForm()
	})
	vBtnRefresh := core.NewButton(vFrame3)
	vBtnRefresh.SetText("Refresh")
	vBtnRefresh.OnClick(func (e events.Event) {
		this.ClearForm()
		this.FormName.SetFocus()
	})

	// database
	this.DbFileName = filepath.Join(GetExecutableFileDirName(), "notes_data.db")
	this.dbSqlite, vErr = sql.Open("sqlite", this.DbFileName)
	if vErr != nil {
		core.MessageDialog(this.Body, "Error connecting database : " + vErr.Error(), "Error")
		return
	}
	this.CreateTables()
	this.Body.SetTitle("My Notes : " + this.DbFileName)
}

func (this *MainForm) CreateTables() {
	var vSql string
	var vErr error

	vSql = `
	CREATE TABLE IF NOT EXISTS notes (
		note_id	INTEGER NOT NULL,
		note_name	TEXT NOT NULL,
		note_data	TEXT NOT NULL,
		add_time	datetime DEFAULT (datetime('now','localtime')),
		update_time	datetime,
		cstat	TEXT NOT NULL DEFAULT ('x')
	);
	
	CREATE TABLE IF NOT EXISTS tags (
		tag_id	INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		note_id	INTEGER NOT NULL,
		tag_name	TEXT NOT NULL,
		cstat	TEXT NOT NULL DEFAULT 'x'
	);
    `
	_, vErr = this.dbSqlite.Exec(vSql)
	if vErr != nil {
		core.MessageDialog(this.Body, vErr.Error(), "Error")
	}
}

func (this *MainForm) OnRemoveClick(e events.Event) {
	funcOkay := func(e events.Event) {
		vErr := this.RemoveNote(this.OldId, nil)
		if vErr != nil {
			core.MessageDialog(this.Body, vErr.Error(), "RemoveNote")
		} else {
			core.MessageDialog(this.Body, "Note Removed", "RemoveNote")
			this.ClearForm()
		}
	}
	ConfirmDialog(this.Body, "Are you sure remove?", funcOkay)
}

func (this *MainForm) SaveForm() {
	var vSql string
	var vNoteId int
	var vSqlTxn *sql.Tx
	var vErr error
	var vNoteName string
	var vNoteContent string
	var vNoteTags string
	var vStr string

	// input validation
	if len(this.FormName.Text()) == 0 {
		core.MessageDialog(this.Body, "Note Name Missing", "Save")
		return
	}

	if len(this.FormContent.Buffer.Text()) == 0 {
		core.MessageDialog(this.Body, "Note Content Missing", "Save")
		return
	}

	if this.OldId == 0 {
		vNoteId = NextId(this.dbSqlite)
	} else {
		vNoteId = this.OldId
	}

	vSqlTxn, vErr = this.dbSqlite.Begin()
	if vErr != nil {
		core.MessageDialog(this.Body, "Not able to start tran : " + vErr.Error())
		return
	}

	if this.OldId > 0 {
		vErr = this.RemoveNote(this.OldId, vSqlTxn)
		if vErr != nil {
			core.MessageDialog(this.Body, "Not able to remove old clData : " + vErr.Error())
			return
		}
	}


	vNoteName = strings.TrimSpace(this.FormName.Text())
	vNoteContent = strings.TrimSpace(string(this.FormContent.Buffer.Text()))
	//vNoteContent = strings.TrimSpace(string(this.FormContent.Text()))
	vNoteTags = strings.TrimSpace(this.FormTag.Text())

	vSql = `
	INSERT INTO NOTES(note_id, note_name, note_data) values(?, ?, ?)
    `
	_, vErr = vSqlTxn.Exec(vSql, vNoteId, vNoteName, vNoteContent)
	if vErr != nil {
		core.MessageDialog(this.Body, "Not able to start tran : " + vErr.Error())
		return
	}

	if len(vNoteTags) > 0  {
		vSql = `INSERT INTO TAGS(note_id, tag_name) values(?, ?)`
		for _, vStr = range strings.Split(vNoteTags, ",") {
			_, vErr = vSqlTxn.Exec(vSql, vNoteId, strings.TrimSpace(vStr))
			if vErr != nil {
				core.MessageDialog(this.Body, "Not able to start tran : " + vErr.Error())
				return
			}
		}
	}

	vErr = vSqlTxn.Commit()
	if vErr != nil {
		core.MessageDialog(this.Body, "Not able to start tran : " + vErr.Error())
		return
	}

	this.ClearForm()
	core.MessageDialog(this.Body, "Data saved", "Save")
}

func (this *MainForm) ClearForm() {
	this.FormName.SetText("")
	this.FormTag.SetText("")
	this.FormContent.Buffer.SetText([]byte(""))
	//this.FormContent.SetText("")
	this.OldId = 0
}

func (this *MainForm) SearchNotes(pStr string) {
	var vSql string
	var vRows *sql.Rows
	var vErr error
	var vId int
	var vName string
	var vTime string

	vSql = `
	select
		distinct a.note_id, a.note_name, datetime(add_time) as add_time
	from
		notes a
		left join tags b on a.note_id = b.note_id and b.cstat='x' 
	where
		a.cstat = 'x'
		and (COALESCE(b.tag_name, '') || a.note_data || a.note_name) like ?
	order by 
		a.note_name
	`
	vRows, vErr = this.dbSqlite.Query(vSql, "%" + pStr + "%")
	if vErr != nil {
		fmt.Println("Error : ", vErr.Error())
		return
	}
	defer vRows.Close()


	// remove old
	this.CardGrid.Clear()

	// add new if any
	for vRows.Next() {
		vRows.Scan(&vId, &vName, &vTime)
		this.CardGrid.Add(vId, vName, vTime)
	}
	this.CardGrid.Update()
}

func (this *MainForm) ShowNote(pId int) {
	var vSql string
	var vRows *sql.Rows
	var vErr error
	var vNoteName string
	var vNoteData string
	var vTags string

	this.ClearForm()

	vSql = `
	select
		a.note_name, a.note_data, group_concat(b.tag_name, ',') as tags
	from
		notes a
		left join tags b on a.note_id = b.note_id and b.cstat='x'
	where
		a.cstat='x'
        and a.note_id = ?
	order by
		b.tag_id
	`
	vRows, vErr = this.dbSqlite.Query(vSql, pId)
	if vErr != nil {
		fmt.Println("Error : ", vErr.Error())
		return
	}
	defer vRows.Close()

	if vRows.Next() == false{
		return
	}

	vRows.Scan(&vNoteName, &vNoteData, &vTags)
	this.FormTag.SetText(vTags)
	this.FormName.SetText(vNoteName)
	this.FormContent.Buffer.SetText([]byte(vNoteData))
	//this.FormContent.SetText(vNoteData)
	this.OldId = pId
}

func (this *MainForm) RemoveNote(pNoteId int, pSqlTxn *sql.Tx) error {
	var vSql string
	var vErr error
	var vSqlTxn *sql.Tx

	if pNoteId <= 0 {
		return errors.New("Please open note first")
	}

	if pSqlTxn == nil {
		vSqlTxn, vErr = this.dbSqlite.Begin()
		if vErr != nil {
			return vErr
		}
	} else {
		vSqlTxn = pSqlTxn
	}

	vSql = `
	UPDATE NOTES 
	SET cstat='e', update_time=datetime(current_timestamp, 'localtime') 
	WHERE cstat='x' and note_id = ? 
	`
	_, vErr = vSqlTxn.Exec(vSql, pNoteId)
	if vErr != nil {
		return vErr
	}

	vSql = `
	UPDATE tags 
	SET CSTAT='e' 
	WHERE NOTE_ID = ? 
	AND CSTAT='x'
	`
	_, vErr = vSqlTxn.Exec(vSql, pNoteId)
	if vErr != nil {
		return vErr
	}

	if pSqlTxn == nil {
		vErr = vSqlTxn.Commit()
		if vErr != nil {
			return vErr
		}
	}

	return nil
}

