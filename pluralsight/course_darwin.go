package pluralsight

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/ajdnik/decrypo/decryptor"
	// sqlite3 driver
	_ "github.com/mattn/go-sqlite3"
)

var (
	unknownCount = -1
	unknownStr   = ""
	ErrNoAuthor  = errors.New("author not found")
)

// CourseRepository fetches video course info from an sqlite database
type CourseRepository struct {
	Path string
}

// getClipsForModule retrieves video clips from an sqlite database that belong to a module
func getClipsForModule(modID int, mod *decryptor.Module, db *sql.DB) error {
	raw, err := db.Query(fmt.Sprintf("select ZTITLE, ZID from ZCLIPCD where ZMODULE=%v order by Z_FOK_MODULE asc", modID))
	if err != nil {
		return err
	}
	defer raw.Close()
	ord := 1
	for raw.Next() {
		var title string
		var id string
		err = raw.Scan(&title, &id)
		if err != nil {
			return err
		}
		clip := decryptor.Clip{
			Order:  ord,
			Title:  title,
			ID:     id,
			Module: mod,
		}
		mod.Clips = append(mod.Clips, clip)
		ord++
	}
	return nil
}

// getAuthorForCourse retrieves author name for a particular course from database
func getAuthorForCourse(cID int, db *sql.DB) (string, error) {
	raw, err := db.Query(fmt.Sprintf("select ZID from ZAUTHORHEADERCD where Z_PK in (select Z_3AUTHORS from Z_3COURSEHEADERS where Z_14COURSEHEADERS=%v)", cID))
	if err != nil {
		return unknownStr, err
	}
	defer raw.Close()
	if !raw.Next() {
		return unknownStr, ErrNoAuthor
	}
	var author string
	err = raw.Scan(&author)
	if err != nil {
		return unknownStr, err
	}
	return author, nil
}

// getModulesForCourse retrieves course modules from an sqlite database that belong to a video course
func getModulesForCourse(cID int, c *decryptor.Course, db *sql.DB) error {
	author, err := getAuthorForCourse(cID, db)
	if err != nil {
		return err
	}
	raw, err := db.Query(fmt.Sprintf("select Z_PK, ZTITLE, ZID from ZMODULECD where ZCOURSE=%v order by Z_FOK_COURSE asc", cID))
	if err != nil {
		return err
	}
	defer raw.Close()
	ord := 1
	for raw.Next() {
		var id int
		var title string
		var uid string
		err = raw.Scan(&id, &title, &uid)
		if err != nil {
			return err
		}
		module := decryptor.Module{
			Order:  ord,
			Title:  title,
			ID:     uid,
			Author: author,
			Clips:  make([]decryptor.Clip, 0),
			Course: c,
		}
		err = getClipsForModule(id, &module, db)
		if err != nil {
			return err
		}
		c.Modules = append(c.Modules, module)
		ord++
	}
	return nil
}

// FindAll finds all of the video courses in the Pluralsight's sqlite database
func (r *CourseRepository) FindAll() ([]decryptor.Course, error) {
	db, err := sql.Open("sqlite3", r.Path)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	raw, err := db.Query("select Z_PK, ZTITLE, ZID from ZCOURSEHEADERCD")
	if err != nil {
		return nil, err
	}
	defer raw.Close()
	courses := make([]decryptor.Course, 0)
	for raw.Next() {
		var id int
		var title string
		var uid string
		err = raw.Scan(&id, &title, &uid)
		if err != nil {
			return courses, err
		}
		course := decryptor.Course{
			Title:   title,
			ID:      uid,
			Modules: make([]decryptor.Module, 0),
		}
		err = getModulesForCourse(id, &course, db)
		if err != nil {
			return courses, err
		}
		courses = append(courses, course)
	}
	return courses, nil
}

// ClipCount returns the number of all video clips in the Pluralsight's database
func (r *CourseRepository) ClipCount() (int, error) {
	db, err := sql.Open("sqlite3", r.Path)
	if err != nil {
		return unknownCount, err
	}
	defer db.Close()
	raw, err := db.Query("select count(*) from ZCLIPCD")
	if err != nil {
		return unknownCount, err
	}
	defer raw.Close()
	if !raw.Next() {
		return unknownCount, sql.ErrNoRows
	}
	var count int
	err = raw.Scan(&count)
	if err != nil {
		return unknownCount, err
	}
	return count, nil
}
