{
    "API": {
        "HTTPHeaders": {
            "Access-Control-Allow-Origin": [
                "http://127.0.0.1:${ipfs_api_port}",
                "https://webui.ipfs.io"
            ],
            "Access-Control-Allow-Methods": [
                "PUT",
                "GET",
                "POST"
            ],
            "Access-Control-Allow-Credentials": true
        }
    },
    "Addresses": {
        "API": "/ip4/127.0.0.1/tcp/${ipfs_api_port}",
        "Announce": [],
        "Gateway": "/ip4/127.0.0.1/tcp/${ipfs_gateway_port}",
        "NoAnnounce": [],
        "Swarm": [
            "/ip4/0.0.0.0/tcp/${ipfs_swarm_port}",
            "/ip6/::/tcp/${ipfs_swarm_port}"
        ]
    },
    "Bootstrap": [],
    "Datastore": {
        "BloomFilterSize": 0,
        "GCPeriod": "1h",
        "HashOnRead": false,
        "Spec": {
            "mounts": [{
                    "child": {
                        "path": "blocks",
                        "shardFunc": "/repo/flatfs/shard/v1/next-to-last/2",
                        "sync": true,
                        "type": "flatfs"
                    },
                    "mountpoint": "/blocks",
                    "prefix": "flatfs.datastore",
                    "type": "measure"
                },
                {
                    "child": {
                        "compression": "none",
                        "path": "datastore",
                        "type": "levelds"
                    },
                    "mountpoint": "/",
                    "prefix": "leveldb.datastore",
                    "type": "measure"
                }
            ],
            "type": "mount"
        },
        "StorageGCWatermark": 90,
        "StorageMax": "${ipfs_storage_max}"
    },
    "Discovery": {
        "MDNS": {
            "Enabled": true,
            "Interval": 10
        }
    },
    "Experimental": {
        "FilestoreEnabled": false,
        "Libp2pStreamMounting": false,
        "P2pHttpProxy": false,
        "QUIC": false,
        "ShardingEnabled": false,
        "UrlstoreEnabled": false
    },
    "Gateway": {
        "APICommands": [],
        "HTTPHeaders": {
            "Access-Control-Allow-Headers": [
                "X-Requested-With",
                "Range",
                "User-Agent"
            ],
            "Access-Control-Allow-Methods": [
                "GET"
            ],
            "Access-Control-Allow-Origin": [
                "*"
            ]
        },
        "NoFetch": false,
        "PathPrefixes": [],
        "RootRedirect": "",
        "Writable": false
    },
    "Ipns": {
        "RecordLifetime": "",
        "RepublishPeriod": "",
        "ResolveCacheSize": 128
    },
    "Mounts": {
        "FuseAllowOther": false,
        "IPFS": "/ipfs",
        "IPNS": "/ipns"
    },
    "Pubsub": {
        "DisableSigning": false,
        "Router": "",
        "StrictSignatureVerification": false
    },
    "Reprovider": {
        "Interval": "12h",
        "Strategy": "all"
    },
    "Routing": {
        "Type": "dht"
    },
    "Swarm": {
        "AddrFilters": null,
        "ConnMgr": {
            "GracePeriod": "20s",
            "HighWater": 900,
            "LowWater": 600,
            "Type": "basic"
        },
        "DisableBandwidthMetrics": false,
        "DisableNatPortMap": false,
        "DisableRelay": false,
        "EnableAutoNATService": false,
        "EnableAutoRelay": false,
        "EnableRelayHop": false
    }
}
