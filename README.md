# KG6WXC MeshMap

[![AREDN](https://kg6wxc.net/images/AREDN_PIR8.png)](https://arednmesh.org)
[![PVARC](https://kg6wxc.net/images/PVARC_Fun.png)](https://www.pvarc.club/)
[![KG6WXC](https://kg6wxc.net/images/WXC_Fun.png)](https://www.kg6wxc.net)

[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)
[![HamRadio](https://img.shields.io/badge/HamRadio-Roger!-green.svg)](https://www.arednmesh.org)
[![MattermostChat](https://img.shields.io/badge/Chat-Mattermost-blueviolet.svg)](https://mattermost.kg6wxc.net/mesh/channels/meshmap)

KG6WXC's updated MeshMap.  
ChatGPT Codex helped a lot with this effort.  
I appreciate everyones support over the years!  
(And some for encouraging me to update this old program)  
This still beta, but works well enough for wider testing.

## New:
One executable for polling, SQLite storage, and the KG6WXC MeshMap webpage.  
SQLite is embedded and is the default: no PHP, C compiler, or database server is needed.  
MariaDB/MySQL remains available, but has yet to be fully tested at this point. [ToDo]  
The map, scripts, styling, logo and Leaflet are embedded; no separate web server or webpage installation is required.  
A reverse proxy is recommended for internet access.  

## Install as a service on Linux

1. Download the binary [release](https://github.com/r1de/KG6WXC-meshmap/releases) file for your system
  
2. Decompress the files into a directory called `meshmap`
    ```bash
    tar -zxvf meshmap-release-file.tgz -C meshmap
    ```
  
3. Enter the `meshmap` directory and create the initial `user-setting.ini` file.
    ```bash
    cd meshmap
    ./meshmap --init
    ```
  
4. Edit the user-settings.ini file according to your setup.
    - Most of the default values should be ok, but they are not good enough!  
    ![Picard Facepalming with the text "No, the defaults are not good enough"](https://kg6wxc.net/images/picard-gh-mm.png)  
    You should go over them and tweak them to your site.  
    Do not forget to change the default map center values or you'll end up looking at So. Cal. Maps.  
    There are many comments trying to describe what each settings does.  
    If you used my original MeshMap, it will look very familiar.  
  
      ```bash
      nano settings/user-settings.ini
      ```
        - Pro Tip: You can just run meshmap as your current user for checking if you got your settings correct.
        Errors will show up in the `logs` directory files.
  
5. Create a `meshmap` system user to run the service as.
    ```bash
    sudo useradd -r -s /bin/false meshmap
    ```
  
6. Now move the meshmap directory that you created to `/opt/meshmap` and set the permissions.
    ```bash
    sudo mv meshmap /opt/
    sudo chown -R meshmap:meshmap /opt/meshmap
    ```
  
7. Edit and move the systemd service file.
    - You shouldn't really have to edit it if you've followed these steps.  
      ```bash
      sudo mv deploy/meshmap.service /etc/systemd/system/
      ```
  
8. Let systemd know about the new service file and start meshmap.  
    ```bash
    sudo systemd daemon-reload
    sudo systemctl enable meshmap
    sudo systemctl start meshmap
    ```
      - meshmap _should_ just start polling if there is not already a database.  
      (this is still Beta remember)  
      Depending on the size of your network, or where you get the initial node list from, it could take up to 20-30 minutes to poll all of the devices.
        - a supernode can easily increase the polling time by magnitudes.
  
9. View the Map  
    - Load up the webpage in your local browser.  
    Unless you used a different port, try clicking [here](http://localhost:8088)

## Build from source

Go 1.23.6 or later is required only on the build machine.  
Dependencies are pinned in go.mod/go.sum.  
SQLite uses a pure-Go driver, so cross-compilation needs no C toolchain.  
Build requires network access once to fetch dependencies.

```bash
git clone https://github.com/r1de/KG6WXC-meshmap.git
cd KG6WXC-meshmap
make build
make release
```

73, KG6WXC
