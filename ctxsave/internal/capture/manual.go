package capture

import (
	"fmt"
	"os"

	"ctxsave/internal/store"
)

func CaptureNote(st *store.Store, project, note string) (*store.Session, error) {
	if note == "" {
		return nil, fmt.Errorf("note cannot be empty")
	}

	sess, err := st.CreateSession("manual", project, "note")
	if err != nil {
		return nil, err
	}

	if _, err := st.AddEntry(sess.ID, store.EntryNote, note, "", 0); err != nil {
		return nil, err
	}

	return sess, nil
}

func CaptureFile(st *store.Store, project, filePath, tag string) (*store.Session, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	content := truncate(string(data), 10000)
	meta := fmt.Sprintf(`{"path":"%s","tag":"%s"}`, filePath, tag)
	label := fmt.Sprintf("file: %s", filePath)

	sess, err := st.CreateSession("file", project, label)
	if err != nil {
		return nil, err
	}

	if _, err := st.AddEntry(sess.ID, store.EntryFile, content, meta, 0); err != nil {
		return nil, err
	}

	return sess, nil
}
