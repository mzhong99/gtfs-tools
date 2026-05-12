GTFS API Record and Playback tool

To develop on this repository, run this command:
```bash
bash <(curl -fsSL "https://raw.githubusercontent.com/mzhong99/gtfs-tools/main/dev-setup.sh?t=$(date +%s)")
```

## pgAdmin

Bring up infrastructure services:

```bash
make infra-up
```

Open pgAdmin:

```text
http://localhost:5050
```

Login credentials:

```text
Email:    admin@example.com
Password: admin
```

Add a PostgreSQL server with the following settings:

```text
Name:             GTFS Local
Host:             host.docker.internal
Port:             5432
Database:         gtfs_db
Username:         gtfs
Password:         gtfs_dev_password
```

`host.docker.internal` is required because PostgreSQL runs in host network mode.
