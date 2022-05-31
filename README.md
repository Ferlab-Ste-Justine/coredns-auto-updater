# About

This is a tool to read fully defined zonefiles under a given etcd prefix path and keep a local directory updated with those zonefiles (creating them, updating them or deleting them as necessary).

The expectation is that each key under the etcd prefix is named after the domain the zonefile is for.

If the above constraint is respected, the tool will be able to update the directory in a way that is compatible with the coredns auto plugin: https://coredns.io/plugins/auto/

Note that the tool watches for changes in the etcd prefix range as opposed to poll for changes and thus, is pretty responsive. Realistically, the reload interval you define for the auto plugin will be the main source of latency for updates.

Also note that the tool expects to be managed by a service manager like systemd to restart on error. The tool does some retries on request errors, but not connection errors and has been well validated and found to be dependable when managed by systemd with restarts as part of this project: https://github.com/Ferlab-Ste-Justine/kvm-coredns-server . Your mileage may vary if you try to run the binary without a monitor to manage restarts.

And finally, note that the tool expects to talk to etcd in a secure manner over a tls connection with either certificate auth or username/password auth.

# Usage

The tool is a binary that can be configured either with a configuration file or environment variables (it tries to look for a **configs.json** file in its running directory and if the file is absent, it fallsback to reading the expected environment variables).

The **configs.json** file is as follows:

```
{
    ZonefilesPath: "Path where zonefiles are to be outputed",
    EtcdKeyPrefix: "Etcd key prefix that the tool will watch on for zonefiles. Suffixes to the prefix should be domain names containing a zonefile",
    EtcdEndpoints: "Command separated list containing entries wwith the format: <ip>:<port>"
    CaCertPath: "Path to the CA certificate that signed the etcd servers' certificates",
    UserAuth: {
        CertPath: "Path to a client certificate. If non-empty,should be accompanied by KeyPath and Username/Password should be empty",
        KeyPath: "Path to a client key",
        Username: "Client username. If non-empty, should be accompanied by by Password and CertPath/KeyPath should be empty",
        Password: "Client password"
    },
    ConnectionTimeout: Connection timeout (number of seconds as integer),
    RequestTimeout: Request timeout (number of seconds as integer),
    RequestRetries: Number of times a failing request should be attempted before exiting on failure, 
}
```

The environment variables are:

- **CONNECTION_TIMEOUT**: Same parameter as **ConnectionTimeout** in **configs.json**
- **REQUEST_TIMEOUT**: Same parameter as **RequestTimeout** in **configs.json**
- **REQUEST_RETRIES**: Same parameter as **RequestRetries** in **configs.json**
- **USER_CERT_PATH**: Same parameter as **UserAuth.CertPath** in **configs.json**
- **USER_KEY_PATH**: Same parameter as **UserAuth.KeyPath** in **configs.json**
- **USER_NAME**: Same parameter as **UserAuth.Username** in **configs.json**
- **USER_PASSWORD**: Same parameter as **UserAuth.Password** in **configs.json**
- **ZONEFILE_PATH**: Same parameter as **ZonefilesPath** in **configs.json**
- **ETCD_ENDPOINTS**: Same parameter as **EtcdEndpoints** in **configs.json**
- **CA_CERT_PATH**: Same parameter as **CaCertPath** in **configs.json**
- **ETCD_KEY_PREFIX**: Same parameter as **EtcdKeyPrefix** in **configs.json**