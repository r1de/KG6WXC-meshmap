# KG6WXC MeshMap
  
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)
[![HamRadio](https://img.shields.io/badge/HamRadio-Roger!-green.svg)](https://www.arednmesh.org)
[![MattermostChat](https://img.shields.io/badge/Chat-Mattermost-blueviolet.svg)](https://mattermost.kg6wxc.net/mesh/channels/meshmap)  
  
Automated mapping of [AREDN](https://arednmesh.org) Networks.  

This is the _new_ KG6WXC MeshMap device polling backend.

2016-2025 - Eric Satterlee / KG6WXC

Licensed under GPL v 3 and later.  

[Donations](https://www.paypal.com/cgi-bin/webscr?cmd=_donations&business=6K5KQYYU34H4U&currency_code=USD&source=url) / Beer Accepted! :-)  

Special thanks and remembrance to Dale, N7QJK (SK) for his help beta testing the original meshmap.  

This README file was contributed by N7CPZ and heavily edited by KG6WXC.  

## Polling Script (Back End) Setup

Project Github: [https://github.com/r1de/KG6WXC-meshmap](https://github.com/r1de/KG6WXC-meshmap)

### Deployment Planning

Despite running an SQL database, this application is very lightweight and doesn't consume many resources.  
It's best deployed in a stand-alone system or an LXC container (or whatever), it will also run on a Ras-Pi. 
>In theory, it _should_ run even on a Windows system.

The amount of backend resources you give it really depends on the size of the network you are going to be polling.  
A Southern California sized network will work on a Ras-Pi, but it will be slow.  
2 CPU cores and 2048 MiB RAM are _more_ than enough to run it smoothly.  
You will *NOT* see a map from this code alone, this only populates the database with info and creates data files for use by the [frontend code](https://github.com/r1de/KG6WXC-meshmap-webpage).


>*If you're upgrading from the original KG6WXC MeshMap, skip to "[Setup for Upgrading an Existing Install](#bkmrk-setup-for-upgrading-)".*


### 1st Time Setup on Debian-Based System

Install required libraries and programs (if not already installed):
- **git**
- **php** (php version 8+ required)
- **php-mysql**
- **php-curl**
- **mariadb-server**
    
```
sudo apt install git php php-mysql php-curl mariadb-server
```  

Clone the project:  
```
sudo git clone https://github.com/r1de/KG6WXC-meshmap.git
```  
Once the cloning operation is complete, the application files will be under a folder called `KG6WXC-meshmap`

>**Hint:** You can also have `git` clone to a directory of your choice:  
>`git clone https://github.com/r1de/KG6WXC-meshmap.git meshmap` - use a directory called "meshmap".  
>`git clone https://github.com/r1de/KG6WXC-meshmap.git .` - use the _current_ directory.

The MariaDB SQL database and user will need to be created before configuring and running the back end for the first time:  
```
sudo mysql
```  
```
CREATE DATABASE node_map;
```  
```
CREATE USER 'mesh-map'@'localhost' IDENTIFIED BY 'your_password_here';
```
```
GRANT ALL PRIVILEGES on node_map.* TO 'mesh-map'@'localhost';
```  
```
FLUSH PRIVILEGES;
```  
```
exit
```  

Enter the project's cloned directory and import the database configuration:  
```
sudo mysql -D node_map < meshmap_db_import.sql
```

Also, copy the default settings template:  
```
cp settings/user-settings.ini-default settings/user-settings.ini
```
>**DO NOT** change the default file, it is used to add new default settings to meshmap.  
>When and if there are new settings you can always override them in your personal settings file.

Edit the _copied_ template for your site:  
```
nano settings/user-settings.ini
```  
This opens the default settings template file, which needs to be modified for your environment.  
I won't list all of the parameters here because then you'll be lazy and won't read the file.  
The file is well-commented and suitably instructional.  
Modify as required, then save as `settings/user-settings.ini`.

---

<details id="bkmrk-setup-for-upgrading-"><summary><strong>Setup for Upgrading an Existing Install</strong></summary>

You likely have everything you need already installed.  
If you have been running the old map and are upgrading to this new version, you will need to update the values in the new configuration file.
>If you copy over your existing user-settings file and carefully update it, you could probably re-use it, there are some changes, but not many.

The Database structure has changed in this new version and it will need to be updated, run this command from the meshmap directory:

```
sudo mysql -D node_map < meshmap_db_update.sql
```
After that is done, run the polling script in test mode and continue with the Front End Setup.

</details>

---

**Note:** You may want to stop here and switch to the [Front End Setup instructions](http://n7cpz-wiki.local.mesh/books/kg6wxc-mesh-map/page/front-end-setup).  
The `pollingScript.php` script uses the `/var/www/html/meshmap/data` directory by default to store the data for the webpage.  
The script will fail if this directory does not exist.  
You can use any other directory (with appropriate write permissions, of course) by changing the `webpageDataDir` value in _your_ `user-settings.ini` file.  
Changing the value to use a temporary directory (say, in your home directory) will allow you to run the following test without having to also setup the front end webpage, which can be done later and you can set the proper directory then.

#### Testing the Polling Script

Still working in the `KG6WXC-meshmap` folder, run:

```
./pollingScript.php --test-mode-with-sql
```

Depending on your account permissions and where exactly you cloned the repository, `sudo` may be required for appropriate write permissions.  
If this is the case, you have done something wrong, `sudo` should **not** be needed (and is **not** recommended by the author)

The output should look something like this, without any warnings or errors:

![Polling Script Example Output](https://kg6wxc.net/misc/pollingScriptTestModeOutput.png)]

#### Configure Scheduled Polling with Cron Job
>**Note:** Set this up _after_ you can run the polling script sucessfully in test mode a couple of times.

`pollingScript.php` will have to run regularly to update the map for changing mesh conditions. Linux uses cron jobs (crontabs) for scheduling recurring tasks. From any directory, run:

```
crontab -e
```

This opens the crontab file where new cron jobs are created and existing jobs can be edited.

To update the map data every half hour:

```
*/30 * * * * /home/your_username_here/KG6WXC-meshmap/pollingScript.php
```

To update every hour at the top of the hour:

```
0 * * * * /home/your_username_here/KG6WXC-meshmap/pollingScript.php
```

To update once daily at midnight:

```
0 0 * * * /home/your_username_here/KG6WXC-meshmap/pollingScript.php
```

Further reading on crontabs and additional options: [https://www.redhat.com/en/blog/linux-cron-command](https://www.redhat.com/en/blog/linux-cron-command)

On exit, you'll see a confirmation message indicating whether a new crontab was created or no changes were made.
Also notice that the cronjob _does not_ use the test-mode switch for the polling script.
Omitting the `--test-mode-with-sql` switch will cause the script to run with no output to the screen except for any errors encountered.  
Depending on the system, `cron` will send any error output, via email, to the user that the `cron` job was run under.


>**Troubleshooting Hint:**
>If the polling script _is_ running from cron and it just does not seem to be working, and you are not getting any messages from `cron` about it, try disabling the cronjob for the time being.
>Just run `crontab -e` again and put a hash mark `#` at the beginning of the meshmap line, then save the file like normal.
>That will disable the job without removing it.
>Then run the polling script with `--test-mode-with-sql` switch and see what the error is.
