package friggdb

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/google/uuid"
)

const (
	workDir = "work"
)

type WAL interface {
	AllBlocks() ([]ReplayBlock, error)
	NewBlock(id uuid.UUID, tenantID string) (HeadBlock, error)
	WorkFolder() string
}

type wal struct {
	c            *walConfig
	workFilepath string
}

type walConfig struct {
	filepath string
}

func newWAL(c *walConfig) (WAL, error) {
	if c.filepath == "" {
		return nil, fmt.Errorf("please provide a path for the WAL")
	}

	// make folder
	err := os.MkdirAll(c.filepath, os.ModePerm)
	if err != nil {
		return nil, err
	}

	workFilepath := path.Join(c.filepath, workDir)
	err = os.RemoveAll(workFilepath)
	if err != nil {
		return nil, err
	}
	err = os.MkdirAll(workFilepath, os.ModePerm)
	if err != nil {
		return nil, err
	}

	return &wal{
		c:            c,
		workFilepath: workFilepath,
	}, nil
}

func (w *wal) AllBlocks() ([]ReplayBlock, error) {
	files, err := ioutil.ReadDir(fmt.Sprintf("%s", w.c.filepath))
	if err != nil {
		return nil, err
	}

	blocks := make([]ReplayBlock, 0, len(files))
	for _, f := range files {
		if f.IsDir() {
			continue
		}

		name := f.Name()
		blockID, tenantID, err := parseFilename(name)
		if err != nil {
			return nil, err
		}

		blocks = append(blocks, &headBlock{
			completeBlock: completeBlock{
				meta:     newBlockMeta(tenantID, blockID),
				filepath: w.c.filepath,
			},
		})
	}

	return blocks, nil
}

func (w *wal) NewBlock(id uuid.UUID, tenantID string) (HeadBlock, error) {
	h := &headBlock{
		completeBlock: completeBlock{
			meta:     newBlockMeta(tenantID, id),
			filepath: w.c.filepath,
		},
	}

	name := h.fullFilename()
	_, err := os.Create(name)
	if err != nil {
		return nil, err
	}

	return h, nil
}

func (w *wal) WorkFolder() string {
	return w.workFilepath
}

func parseFilename(name string) (uuid.UUID, string, error) {
	i := strings.Index(name, ":")

	if i < 0 {
		return uuid.UUID{}, "", fmt.Errorf("unable to parse %s", name)
	}

	blockIDString := name[:i]
	tenantID := name[i+1:]

	blockID, err := uuid.Parse(blockIDString)
	if err != nil {
		return uuid.UUID{}, "", err
	}

	return blockID, tenantID, nil
}
