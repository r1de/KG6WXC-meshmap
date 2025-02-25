# Back End Setup

Project Github: [https://github.com/r1de/KG6WXC-meshmap](https://github.com/r1de/KG6WXC-meshmap)

### Deployment Planning

Despite running an SQL database, this application is very lightweight and doesn't consume many resources.  
It's best deployed with stand-alone system or an LXC container (or whatever), it will also run on a Ras-Pi.  
The amount of backend resources you give it really depends on the size of the network you are going to be polling.  
A Southern California sized network will work on a Ras-Pi, but it will be slow.  
2 CPU cores and 2048 MiB RAM are _more_ than enough to run it smoothly.  
You will *NOT* see a map from this code alone, this only populates the database with info and creates data files for use by the [frontend code](https://github.com/r1de/KG6WXC-meshmap-webpage).


>*If you're upgrading from the original KG6WXC MeshMap, skip to "[Setup for Upgrading an Existing Install](#setup-for-existing-install)".*


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

>**Hint:** You can also get have it copy to a directory of your choice:  
>`git clone https://github.com/r1de/KG6WXC-meshmap.git meshmap` - copy files to a directory called "meshmap".  
>`git clone https://github.com/r1de/KG6WXC-meshmap.git .` - copy the files to the _current_ directory.

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


</details><details id="bkmrk-setup-for-upgrading-"><summary>Setup for Upgrading an Existing Install</summary>

#### Setup for Existing Install

If you have been running this code and you have errors about things missing in the Database, run the following command with the update file:

```
sudo mysql -D node_map < meshmap_db_update.sql
```
</details>

<p class="callout warning">**Stop here and switch to the [Front End Setup instructions](http://n7cpz-wiki.local.mesh/books/kg6wxc-mesh-map/page/front-end-setup).** The `pollingScript.php` script relies on the `/var/www/html/meshmap/data` directory existing and will fail unless it's created during the front end setup process.</p>

#### Testing

Still working in the `KG6WXC-meshmap/settings` folder, run:

```
sudo ./pollingScript.php --test-mode-with-sql
```

Depending on your account permissions and where exactly you cloned the repository, `sudo` may be required for appropriate write permissions.

The output should look something like this, without any warnings or errors:

[![image.png](http://n7cpz-wiki.local.mesh/uploads/images/gallery/2025-02/scaled-1680-/0FFimage.png)](http://n7cpz-wiki.local.mesh/uploads/images/gallery/2025-02/0FFimage.png)

#### Configure Scheduled Polling with Cron Job

`pollingScript.php` will have to run regularly to update the map for changing mesh conditions. Linux uses cron jobs (crontabs) for scheduling recurring tasks. From any directory, run:

```
sudo crontab -e
```

This opens the crontab file where new cron jobs are created and existing jobs can be edited. `sudo` here ensures the job will run with required permissions to write in the default `/home/your_username_here/` directory. Otherwise you may encounter write privilege errors.

To update the map data every half hour:

```
*/30 * * * * /home/your_username_here/KG6WXC-meshmap/pollingScript.php
```

To update every hour at the top of the hour:

```
0 * * * * /home/your_username_here/KG6WXC-meshmap/pollingScript.php
```

To update once daily:

```
0 0 * * * /home/your_username_here/KG6WXC-meshmap/pollingScript.php
```

Further reading on crontabs and additional options: [https://www.redhat.com/en/blog/linux-cron-command](https://www.redhat.com/en/blog/linux-cron-command)

On exit, you'll see a confirmation message indicating whether a new crontab was created or no changes were made. You'll want to verify whether the job was created correctly by checking for cron log errors. There are two good methods for doing this: either run `sudo grep CRON /var/log/syslog` directly, or by checking the mail messages (cron will "email" a user running jobs with errors) under `/var/mail/`.

(this file was contributed by N7CPZ and edited by KG6WXC)
