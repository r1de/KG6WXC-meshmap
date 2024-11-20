# KG6WXC-meshmap

[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)
[![HamRadio](https://img.shields.io/badge/HamRadio-Roger!-green.svg)](https://www.arednmesh.org)
[![MattermostChat](https://img.shields.io/badge/Chat-Mattermost-blueviolet.svg)](https://mattermost.kg6wxc.net/mesh/channels/meshmap)  
Automated mapping of [AREDN](https://arednmesh.org) Networks.  

This is the _new_ KG6WXC MeshMap device polling backend.  
It is _very much_ a work in progress, and is being actively worked on, but it _does_ actually work.  
Instructions have not been written yet, setup is very similar to the original meshmap.  
  
You will _not_ see a map webpage from this, it is only the part that populates the database and will output some java files for use _with_ the webpage.  
Webpage code is here:  
https://github.com/r1de/KG6WXC-meshmap-webpage  
  
## REQUIREMENTS
- **php**
- **php-mysql**
- **php-curl**
- **mariadb-server**

If you want to run this, the MariaDB database will need to be setup before hand.  
  
If you had the previous version of MeshMap, update the database from the new sql file:  
`sudo mysql -D node_map < meshmap_db_import.sql`  

For now, updates to the Database will be in the `meshmap_db_update.sql` file.  
If you have been running this code and you have errors about things missing in the Database, please run the above command with the update file.  
`sudo mysql -D node_map < meshmap_db_update.sql`  
*note: this is NOT needed for a new setup.

If you were not running the previous version setup the SQL database like this _and then_ run the above command to import the sql file:  
> `sudo mysql`  
> `CREATE DATABASE node_map;`  
> `CREATE USER 'mesh-map'@'localhost' IDENTIFIED BY 'password';`  
> `GRANT ALL PRIVILEGES on node_map.* TO 'mesh-map'@'localhost';`  
> `FLUSH PRIVILEGES;`  

Copy `settings/user-settings.ini-default` to `settings/user-settings.ini` and edit for your site.  

Manually run `pollingScript.php --test-mode-with-sql` to make sure everything is setup properly and you get no errors.

To run automatically use cron (this example will poll every 30 min):  
`*/30 * * * * /home/kg6wxc/KG6WXC-meshmap/pollingScript.php`

SQLite is not fully working yet, don't try to use it.  
  
Plz contact [KG6WXC](mailto:kg6wxc@gmail.com?subject=MeshMap%20Help) for help if you wish to help test this or have questions.  
  
Any help is appreciated! 
  
73 DE KG6WXC
