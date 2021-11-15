package filesystem

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
)

func EnsureZonefilesDir(zonefilesPath string) error {
	_, err := os.Stat(zonefilesPath)
	if err != nil &&  errors.Is(err, os.ErrNotExist) {
		mkErr := os.MkdirAll(zonefilesPath, os.ModePerm)
		if mkErr != nil {
			return errors.New(fmt.Sprintf("Error creating zonefiles directory: %s", mkErr.Error()))
		}
	}

	return nil
}

func ListZonefiles(zonefilesPath string) ([]string, error) {
	zonefiles, err := ioutil.ReadDir(zonefilesPath)
	if err != nil {
		return []string{}, errors.New(fmt.Sprintf("Error listing zonefiles: %s", err.Error()))
	}

	result := make([]string, len(zonefiles))
	for idx, zonefile := range zonefiles {
		result[idx] = zonefile.Name()
	}
	return result, nil
}

func GetZonefileDeletions(newZonefiles map[string]string, PreExistingZonefiles []string) []string {
	deletions := []string{}
	for _, zonefile := range PreExistingZonefiles {
		if _, ok := newZonefiles[zonefile]; !ok {
			deletions = append(deletions, zonefile)
		}
	}

	return deletions
}

func DeleteZonefile(zonefilesPath string, zonefile string) error {
	err := os.Remove(path.Join(zonefilesPath, zonefile))
	if err != nil {
		return errors.New(fmt.Sprintf("Error deleting zonefile: %s", err.Error()))
	}

	return nil
}

func UpsertZonefile(zonefilesPath string, zonefile string, content string) error {
	err := ioutil.WriteFile(path.Join(zonefilesPath, zonefile), []byte(content), 0644)
	if err != nil {
		return errors.New(fmt.Sprintf("Error upserting zonefile: %s", err.Error()))
	}

	return nil
}

func ApplyZonefilesChanges(zonefilesPath string, upsertedZonefiles map[string]string, deletedZonefiles []string) error {
	for k, v := range upsertedZonefiles {
		err := UpsertZonefile(zonefilesPath, k, v)
		if err != nil {
			return err
		}
	}

	for _, v := range deletedZonefiles {
		err := DeleteZonefile(zonefilesPath, v)
		if err != nil {
			return err
		}
	}

	return nil
}