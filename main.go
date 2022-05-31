package main

import (
    "fmt"
    "os"

	"ferlab/coredns-auto-updater/configs"
	"ferlab/coredns-auto-updater/etcd"
	"ferlab/coredns-auto-updater/filesystem"
)

func syncZonefiles() error {
	confs, err := configs.GetConfigs()
	if err != nil {
		return err
	}

	filesystem.EnsureZonefilesDir(confs.ZonefilesPath)

	cli, connErr := etcd.Connect(
		confs.UserAuth.CertPath, 
		confs.UserAuth.KeyPath, 
		confs.UserAuth.Username,
		confs.UserAuth.Password,
		confs.CaCertPath, 
		confs.EtcdEndpoints, 
		confs.ConnectionTimeout, 
		confs.RequestTimeout, 
		confs.RequestRetries,
	)
	if connErr != nil {
		return connErr	
	}
	defer cli.Client.Close()

	existingZonefiles, exZnfsErr := filesystem.ListZonefiles(confs.ZonefilesPath)
	if exZnfsErr != nil {
		return exZnfsErr
	}

	zonefiles, rev, znfsErr := cli.GetZonefiles(confs.EtcdKeyPrefix)
	if znfsErr != nil {
		return znfsErr
	}

	zonefileDeletions := filesystem.GetZonefileDeletions(zonefiles, existingZonefiles)

	applyErr := filesystem.ApplyZonefilesChanges(confs.ZonefilesPath, zonefiles, zonefileDeletions)
	if applyErr != nil {
		return applyErr
	}

	events := make(chan etcd.ZonefileEvent)
	go cli.WatchZonefiles(confs.EtcdKeyPrefix, rev, events)

	for e := range events {
		if e.Err != nil {
			return e.Err
		}

		if e.Action == "upsert" {
			err := filesystem.UpsertZonefile(confs.ZonefilesPath, e.Domain, e.Content)
			if err != nil {
				return err
			}
		} else {
			err := filesystem.DeleteZonefile(confs.ZonefilesPath, e.Domain)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func main() {
	err := syncZonefiles()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}