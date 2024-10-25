#!/usr/bin/env php
<?php
//only run from the command line
if (PHP_SAPI !== 'cli') {
	$file = basename($_SERVER['PHP_SELF']);
	exit("<style>html{text-align: center;}p{display: inline;}</style>
        <br><strong>This script ($file) should only be run from the
        <p style='color: red;'>command line</p>!</strong>
        <br>exiting...");
}
//get start time
$mtimeStart = microtime(true);

/************************************
* New polling script for meshmap apr 2021-2024 - kg6wxc
* Original meshmap scripts are from 2016 and beyond.
* 
* Licensed under GPLv3 or later
* This script is the heart of kg6wxcs' mesh map system.
* bug fixes, improvements and corrections are welcomed!
*
* This file is part of the Mesh Mapping System.
* The Mesh Mapping System is free software: you can redistribute it and/or modify
* it under the terms of the GNU General Public License as published by
* the Free Software Foundation, either version 3 of the License, or
* (at your option) any later version.
*
* The Mesh Mapping System is distributed in the hope that it will be useful,
* but WITHOUT ANY WARRANTY; without even the implied warranty of
* MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
* GNU General Public License for more details.
*
* You should have received a copy of the GNU General Public License
* along with The Mesh Mapping System.  If not, see <http://www.gnu.org/licenses/>.
************************************/

$INCLUDE_DIR = dirname(__FILE__);
require $INCLUDE_DIR . "/include/pollingFunctions.inc";
require $INCLUDE_DIR . "/include/mysqlFunctions.inc";
require $INCLUDE_DIR . "/include/colorizeOutput.inc";
require $INCLUDE_DIR . "/include/outputToConsole.inc";
require $INCLUDE_DIR . "/include/outputToFile.inc";
require $INCLUDE_DIR . "/include/checkArednVersions.inc";
require $INCLUDE_DIR . "/include/sqliteStuff.inc";
require $INCLUDE_DIR . "/include/calcDistanceAndBearing.inc";
require $INCLUDE_DIR . "/include/createJS.inc";
require $INCLUDE_DIR . "/include/node_report_data.inc";

$USE_SQL = 1;
$TEST_MODE = 0;

if (!empty($argv[1])) {
	switch ($argv[1]) {
		case "--help":
			echo $argv[0] . " Usage:\n\n";
			echo $argv[1] . "\tThis help message\n\n";
			echo "--data-files-only\tNo polling, only regenerate the data files used by the webpage\n";
			echo wxc_addColor("this option does not work yet", "redBold") . "\n\n";
			echo "--test-mode-no-sql\tDO NOT access database only output to screen\n";
			echo "(or to the log files when using parallel mode)\n";
			echo "(this parameter is useful for testing)\n\n";
			echo "--test-mode-with-sql\tDO access the database AND output to screen\n";
			echo "(useful to see if everything is working and there are no errors reading/writing to the database)\n\n";
//			echo "No arguments to this script will run it in \"silent\" mode, good for cron jobs! :)\n";
			echo "\n";
			exit();
	    case "--test-mode-no-sql":
			$USE_SQL = 0;
			$TEST_MODE = 1;
			break;
	    case "--test-mode-with-sql":
	    	$TEST_MODE = 1;
			break;
	    case "--data-files-only":
	    	$USE_SQL = 0;
	    	$START_POLLING = 0;
	    	break;
	    default:
	    	exit("Unknown parameter\nTry: " . $argv[0] . wxc_addColor(" --help", "red") . "\n");	    	
	}
}

global $USER_SETTINGS;

//load defaults into $USER_SETTINGS
if (file_exists($INCLUDE_DIR . "/settings/user-settings.ini-default")) {
	$DEFAULT_USER_SETTINGS = parse_ini_file($INCLUDE_DIR . "/settings/user-settings.ini-default");
}else {
	exit("settings/user-settings.ini-default is missing!\n\n");
}

//check for users custom settings
if (file_exists($INCLUDE_DIR . "/settings/user-settings.ini")) {
	$USER_SETTINGS = parse_ini_file($INCLUDE_DIR . "/settings/user-settings.ini");
}else {
	exit("\n\nYou **must** copy the user-settings.ini-default file to user-settings.ini and edit it!!\n\n");
}

//merge default settings and users custom settings
$USER_SETTINGS = array_merge($DEFAULT_USER_SETTINGS, $USER_SETTINGS);

//first grab the localnodes actual name
//this is used to locate the map data "origin"
if($TEST_MODE) {
	echo "Polling " . $USER_SETTINGS['localnode'] . " for some info before starting... ";
}
$localInfo = @file_get_contents("http://" . $USER_SETTINGS['localnode'] . "/cgi-bin/sysinfo.json");
if($localInfo == "" || is_null($localInfo)) {
	if(!isset(error_get_last()['message']) || is_null(error_get_last()['message']) || error_get_last()['message'] == "") {
		exit("Could not get map origin node name: No error, just... nothing, null, nada.");
	}else {
		$failReason = trim(substr(strrchr(error_get_last()['message'], ":"), 1));
		//AREDN port 8080 is going away, try new way
		if($failReason === "Connection refused") {
			$localInfo = "";
			$localInfo = @file_get_contents("http://" . $USER_SETTINGS['localnode'] . ":8080/cgi-bin/sysinfo.json");
			if($localInfo == "" || is_null($localInfo)) {
				if(!isset(error_get_last()['message']) || is_null(error_get_last()['message']) || error_get_last()['message'] == "") {
					exit("Could not get map origin node name: No error, just... nothing, null, nada.");
				}else {
					exit("Could not get map origin node name: " . trim(substr(strrchr(error_get_last()['message'], ":"), 1)));
				}
			}
		}else {
			exit("Could not get map origin node name: " . trim(substr(strrchr(error_get_last()['message'], ":"), 1)));
		}
	}
	
}
$localInfo = preg_replace('/[\x00-\x1F\x7F-\xFF]/', '', $localInfo);
$localInfo = json_decode($localInfo,true);
$localNodeName = $localInfo['node'];
unset($localInfo);
if($TEST_MODE) {
	echo wxc_addColor("Done!", "greenBold") . "\n";
}

//TODO: combine this with above check for node name and make into a function...  seriously! This is getting messy again.
if($TEST_MODE) { 
	echo "Attempting to retrieve network topology from " . $USER_SETTINGS['localnode'] . "... ";
}
//attempt to get network topology from AREDN API
//try the new way (any firmware after 10-17-2024)
$allMeshNodes = @file_get_contents("http://" . $USER_SETTINGS['localnode'] . "/cgi-bin/sysinfo.json?topology=1");
//if that doesn't work try the old way (pre 10-17-2024)
if (!strpos($allMeshNodes, "topology")) {
	$allMeshNodes = "";
	$allMeshNodes = @file_get_contents("http://" . $USER_SETTINGS['localnode'] . ":8080/cgi-bin/api?mesh=topology");
	if(empty($allMeshNodes)) {
		//the dang node did not return anything, they do this sometimes, try again after a 10 seconds
		if($TEST_MODE) {
			echo "\nTrying again after 10 seconds... ";
		}
		sleep(10);
		//try the new way (any firmware after 10-17-2024)
		$allMeshNodes = @file_get_contents("http://" . $USER_SETTINGS['localnode'] . "/cgi-bin/sysinfo.json?topology=1");
		//if that doesn't work try the old way (pre 10-17-2024)
		if (!strpos($allMeshNodes, "topology")) {
			$allMeshNodes = "";
			$allMeshNodes = @file_get_contents("http://" . $USER_SETTINGS['localnode'] . ":8080/cgi-bin/api?mesh=topology");
		}
		if(!strpos($allMeshNodes, "topology") || empty($allMeshNodes)) {
			//we have failed 2x, just exit, something is wrong
			exit(wxc_addColor("\nTHERE WAS A PROBLEM ACCESSING THE API ON YOUR LOCALNODE!\n" . error_get_last()['message'] . "\n", "redBold"));
		}
	}
}

//decode the json retrieved from localnode
$allMeshNodes = @json_decode($allMeshNodes, true);
if (!is_array($allMeshNodes)) {
	if($TEST_MODE) { echo "\n"; }
	exit(wxc_addColor("THERE WAS A PROBLEM DECODING THE TOPOLOGY JSON FROM YOUR LOCALNODE!\n JSON_ERR_MSG: " . json_last_error_msg(), "redBold")  . "\n\n");
}

//pull out just the topology info and clear old $var.
if (@is_array($allMeshNodes['pages']['mesh']['topology']['topology'])) {
	$topoInfo = $allMeshNodes['pages']['mesh']['topology']['topology'];
}elseif(@is_array($allMeshNodes['topology'])) {
        $topoInfo = $allMeshNodes['topology'];
}else {
	if($TEST_MODE) { echo "\n"; }
	exit(wxc_addColor("YOUR LOCALNODES' API DOES NOT CONTAIN NETWORK TOPOLOGY INFORMATION!", "redBold")  . "\n\n");
}
unset($allMeshNodes);
if($TEST_MODE) {
	echo wxc_addColor("Done!", "greenBold") . "\n";
}

//use the 'lastHopIP' to build new array with the info for each device
//using lastHopIP will ensure a unique list of nodes on the network
//add in the "hopsAway" variable (this would be hops away from the localnode)
if($TEST_MODE) {
	echo "Building list of node IP addresses and some link info... ";
}
$nodeDevices = [];
$MaxNumHops = 0;

foreach($topoInfo as $link) {
	$nodeDevices[$link['lastHopIP']] = [];
	$nodeDevices[$link['lastHopIP']]['hopsAway'] = $link['hops'];
	if(intval($link['hops']) > $MaxNumHops) {
		$MaxNumHops = intval($link['hops']);
	}
}
unset($link);

//build up the $nodeDevices array with info we need about each link
foreach($nodeDevices as $ip => $node) {
	foreach($topoInfo as $link) {
		if($link['lastHopIP'] == $ip) {
			$nodeDevices[$ip]['link_info'][$link['destinationIP']] = $link;
		}
	}
	unset($link);
	unset($ip);
	unset($node);
}
unset($topoInfo);
if($TEST_MODE) {
	echo wxc_addColor("Done!", "greenBold") . "\n";
}


//get a total count of number of ip's to poll
$TotalToPoll = count($nodeDevices);


//clear the log files from the last run
if($TEST_MODE) {
	echo "Clearing log files... ";
}
$logFile = fopen($INCLUDE_DIR . "/logs/polling_output.log", "w");
fwrite($logFile, "### THIS LOG FILE IS RECREATED EACH TIME THE POLLING SCRIPT RUNS.\n### ANY CHANGES WILL BE LOST.\n");
fwrite($logFile, "### LAST RUN: " . date("M j G:i:s T (P)") . "\n\n");
fclose($logFile);

$err_log_file = fopen($INCLUDE_DIR . "/logs/polling_errors.log", "w");
fwrite($err_log_file, "### THIS LOG FILE IS RECREATED EACH TIME THE POLLING SCRIPT RUNS.\n### ANY CHANGES WILL BE LOST.\n");
fwrite($err_log_file, "### LAST RUN: " . date("M j G:i:s T (P)") . "\n\n");
fclose($err_log_file);

$noLoc = fopen($INCLUDE_DIR . "/logs/no_location.log", "w");
fwrite($noLoc, "### THIS LOG FILE IS RECREATED EACH TIME THE POLLING SCRIPT RUNS.\n### ANY CHANGES WILL BE LOST.\n");
fwrite($noLoc, "### LAST RUN: " . date("M j G:i:s T (P)") . "\n\n");
fclose($noLoc);
if($TEST_MODE) {
	echo wxc_addColor("Done!", "greenBold") . "\n";
}
$dbHandle = "";

//use $nodeDevices array to start populating the database.
//$nodeDevice looks like this
//[Mesh Node IP]
//  [hopsAway from the "localnode"]
//  [some info about how your localnode sees the other nodes on the network]
//
if($TEST_MODE) {
	echo "Connecting to " . $USER_SETTINGS['SQL_TYPE'] . "... ";
}
if($USE_SQL && $USER_SETTINGS['SQL_TYPE'] == "sqlite") {
	if(!file_exists($INCLUDE_DIR . "/sqlite3_db/mesh-map.sqlite")) {
		$dbHandle = create_sqlite_db($INCLUDE_DIR . "/sqlite3_db/mesh-map.sqlite");
	}else {
		$dbHandle = new SQLite3($INCLUDE_DIR . "/sqlite3_db/mesh-map.sqlite");
	}
	foreach($nodeDevices as $wlan_ip => $info) {
		$sql = "REPLACE INTO 'node_info' ('wlan_ip', 'hopsAway', 'link_info') VALUES('" .
				$wlan_ip . "', " .
				$info['hopsAway'] . ", " .
				escapeshellarg(serialize($info['link_info'])) . ")";
		$dbHandle->exec($sql);
	}
	$dbHandle->close();
}
if($USE_SQL && $USER_SETTINGS['SQL_TYPE'] == "mysql") {
	$sql_connection = wxc_connectToMySQL();
	//do stuff here
	foreach($nodeDevices as $wlan_ip => $info) {
		$sql = "INSERT INTO node_info (wlan_ip, hopsAway, link_info) VALUES ('" .
				$wlan_ip . "', " . $info['hopsAway'] . ", " .
				escapeshellarg(serialize($info['link_info'])) . ") ON DUPLICATE KEY UPDATE " .
				"hopsAway = " . $info['hopsAway'] . ", " .
				"link_info = " . escapeshellarg(serialize($info['link_info']));
				escapeshellarg(serialize($info['link_info'])) . ")";
		wxc_putMySql($sql_connection, $sql);
	}
	mysqli_close($sql_connection);
}
if($TEST_MODE) {
	echo wxc_addColor("Done!", "greenBold") . "\n";
}


//TODO: move to settings
$autoCheckArednVersions = true;

if($autoCheckArednVersions) {
	if($TEST_MODE) {
		echo "Checking Internet for the latest AREDN version numbers... ";
	}
	$versions = get_current_aredn_versions();
	if($TEST_MODE) {
		echo wxc_addColor("Done!", "greenBold") . "\n(Stable: " . $versions['AREDN_STABLE_VERSION'] . " Nightly: " . $versions['AREDN_NIGHTLY_VERSION'] . ")\n\n";
	}

	if($USE_SQL) {
		$sql = "INSERT INTO aredn_info (id, current_stable_version, current_nightly_version) VALUES('AREDNINFO', '" .
				$versions['AREDN_STABLE_VERSION'] . "', '" .
				$versions['AREDN_NIGHTLY_VERSION'] . "') ON DUPLICATE KEY UPDATE " .
				"current_stable_version = '" . $versions['AREDN_STABLE_VERSION'] . "', " .
				"current_nightly_version = '" . $versions['AREDN_NIGHTLY_VERSION'] . "'";

		if($USER_SETTINGS['SQL_TYPE'] == "sqlite") {
			$dbHandle = new SQLite3($INCLUDE_DIR . "/sqlite3_db/mesh-map.sqlite");
			$dbHandle->exec($sql);
			$dbHandle->close();
		}elseif($USER_SETTINGS['SQL_TYPE'] == "mysql") {
			$sql_connection = wxc_connectToMySQL();
			wxc_putMySql($sql_connection, $sql);
			mysqli_close($sql_connection);
		}
	}

	//also save to a file for parallel script
	$version_file = fopen($INCLUDE_DIR . "/logs/latest_versions.txt", "w");
	fwrite($version_file, $versions['AREDN_STABLE_VERSION'] . "\n");
	fwrite($version_file, $versions['AREDN_NIGHTLY_VERSION'] . "\n");
	fclose($version_file);

	
}else {
	//TODO: don't autocheck, use the value from the config file from stable FW version.
}


$mysql_db = "";
$sqlite3_db = "";

$nodeCount = 0;

$START_POLLING = 1;

//lets go polling!
if($START_POLLING) {
	$donePolling = 0;

	//TODO: DO NOT LEAVE IT LIKE THIS
	$USER_SETTINGS['node_polling_parallel'] = "true";

	if($USER_SETTINGS['node_polling_parallel']) {
		if($TEST_MODE) {
			echo "Polling Progress (" . $USER_SETTINGS['numParallelThreads'] . "x): ";
		}
		$numParallelProcesses = $USER_SETTINGS['numParallelThreads'];
		$pProcessingChunks = array_chunk($nodeDevices, 1, true);
		unset($nodeDevices);
		$pProcessingPIDS = [];
		for($i = 0; $i < count($pProcessingChunks); $i++) {
			$chunk = $pProcessingChunks[$i];
			foreach($chunk as $ip => $info) {
				$ipExtraInfo = escapeshellarg(serialize($info));
				$pProcessingPIDS[] = exec("php ". $INCLUDE_DIR . "/parallel/parallelPolling.php " . $ip . " " . $USE_SQL . " >> " . $INCLUDE_DIR . "/logs/polling_output.log & echo $!");
				$nodeCount++;
			}
			while(count($pProcessingPIDS) > $numParallelProcesses) {
				foreach($pProcessingPIDS as $index => $pid) {
					if(!file_exists("/proc/$pid")) {
						unset($pProcessingPIDS[$index]);
						$donePolling++;
					$percent = floor(($donePolling / $TotalToPoll) * 100);
					$numLeft = 100 - $percent;
					if($TEST_MODE) {
						printf("\033[26G%u%% (%u/%u)... ", $percent, $donePolling, $TotalToPoll);
					}
					//echo $progress;
					}
				}
			}
		}
		
		//wait for all scripts to finish so we can see how long it actually takes
		while(count($pProcessingPIDS) > 0) {
			foreach($pProcessingPIDS as $index => $pid) {
				if(!file_exists("/proc/$pid")) {
					unset($pProcessingPIDS[$index]);
					$donePolling++;
					$percent = floor(($donePolling / $TotalToPoll) * 100);
					$numLeft = 100 - $percent;
					if($TEST_MODE) {
						printf("\033[26G%u%% (%u/%u)... ", $percent, $donePolling, $TotalToPoll);
					}
				}
			}
		}
		if($TEST_MODE) {
			echo wxc_addColor("Done!", "greenBold") . "\n";
		}
	}
}

//connect back to SQL
$sql_connection = wxc_connectToMySQL();

//find the supernodes and treat them as we did our localnode above.
$superNodes = wxc_getMysqlFetchAll("select wlan_ip from node_info where mesh_supernode = 'true'");
//foreach($superNodes as $super) {

//	}
//var_dump($superNodes);
//exit();
//TODO

//create the topology info
if($TEST_MODE) {
	echo "Building Topology information: ";
}
$link_count = 0;

$query = wxc_getMysqlFetchAll("select node from node_info");

foreach($query as $v) {
	$node = $v['node'];

	$query = "SELECT node, lat, lon, link_info from node_info where node like '" . $v['node'] . "' and (lat != '0' || lon != '0')";
	$q_results = wxc_getMySql($query);
	if(isset($q_results['link_info'])) {
		$links = unserialize($q_results['link_info']);
	}

	if(!empty($links)) {
		foreach($links as $k => $v){
			$query = "SELECT lat, lon from node_info where wlan_ip = '" . $k  . "' and (lat != '0' || lon != '0')";
			$link_coords = wxc_getMySql($query);
	
			if(isset($q_results['lat']) && isset($q_results['lon']) && isset($link_coords['lat']) && isset($link_coords['lon'])) {
				if(!empty($q_results['lat']) && !empty($q_results['lon']) && !empty($link_coords['lat']) && !empty($link_coords['lon'])) {
					$links[$k]['linkLat'] = $link_coords['lat'];
					$links[$k]['linkLon'] = $link_coords['lon'];
	
					if(isset($links[$k]['linkType'])) {
						if($links[$k]['linkType'] == "RF") {
							$dist_bear = wxc_getDistanceAndBearing($q_results['lat'], $q_results['lon'], $link_coords['lat'], $link_coords['lon']);
							$links[$k]['distanceKM'] = $dist_bear['distanceKM'];
							$links[$k]['distanceMiles'] = $dist_bear['distanceMiles'];
							$links[$k]['bearing'] = $dist_bear['bearing'];
						}
					}

					$link_count++;
					unset($v);
					if($TEST_MODE) {
						printf("\033[32G(%u)... ", $link_count);
					}	
				}
			}
		}
		$update_link_info = "update node_info set link_info = " . escapeshellarg(serialize($links)) . " where node = '" . $node . "'";
		wxc_putMySql($sql_connection, $update_link_info);
	}
}
if($TEST_MODE) {
	echo wxc_addColor("Done!", "greenBold");
	echo "\n\n";
}

//get polling stats and echo in test mode
$pollingInfo = [];

//$pollingInfo['id'] = "POLLINFO";
$pollingInfo['numParallelThreads'] = $USER_SETTINGS['numParallelThreads'];

$pollingInfo['nodeTotal'] = $nodeCount;
if ($TEST_MODE) {
	//total nodes found to try and poll
	echo "Total Node Count: " . $nodeCount . "\n";
}
	//nodes that returned garbage
	$fCount = 0;
	$f = $INCLUDE_DIR . "/logs/polling_errors.log";
	$h = fopen($f, "r");
	while(!feof($h)) {
		$line = fgets($h);
		if (strpos($line, "#") !== false || empty(trim($line))) {
			continue;
			//|| !preg_match('/^#/', $line) || !preg_match('/^\n/', $line)) {
			//$fCount++;
		}else {
			$fCount++;
		}
	}
	fclose($h);

$pollingInfo['garbageReturned'] = $fCount;
if($TEST_MODE) {
	echo "Garbage Returned: " . $fCount . "\n";
}
$pollingInfo['highestHops'] = $MaxNumHops;

if($TEST_MODE) {
	echo "Highest Hops Away: " . $MaxNumHops . "\n";
}
	//total nodes found minus ones that return garbage
	$totalPolled = intval($nodeCount) - intval($fCount);
$pollingInfo['totalPolled'] = $totalPolled;
if($TEST_MODE) {
	echo "Total Nodes polled: " . $totalPolled . "\n";
}

	$fCount = 0;
	$f = $INCLUDE_DIR . "/logs/no_location.log";
	$h = fopen($f, "r");
	while(!feof($h)) {
		$line = fgets($h);
		if (strpos($line, "#") !== false || empty(trim($line))) { 
			continue;
// || strpos($line, "\n") !== false) {
//		if (!preg_match('/^#/', $line) || !preg_match('/^\n/', $line)) {
			//$fCount++;
		}else {
			$fCount++;
		}
	}
	fclose($h);
$mapTotal = intval($totalPolled) - intval($fCount);
$pollingInfo['noLocation'] = $fCount;
$pollingInfo['mappableNodes'] = $mapTotal;
if($TEST_MODE) {
	echo "Nodes with no location: " . $fCount . "\n";
	echo "Total that can be shown on map: " . $mapTotal . "\n";
}
$pollingInfo['mappableLinks'] = $link_count;
if($TEST_MODE) {
	echo "Links found that can be mapped: " . $link_count . "\n";
}
	//total time taken to run the polling
	$mtimeEnd = microtime(true);
	$totalTime = $mtimeEnd-$mtimeStart;
$pollingInfo['pollingTimeSec'] = round($totalTime, 2);
if($TEST_MODE) {
	//display how long it took to poll all the nodes
	echo "Time Elapsed: " . round($totalTime, 2) . " seconds ( " . round($totalTime/60, 2) . " minutes ).\n\n";
}

$q = "INSERT INTO map_info (id, ";
foreach($pollingInfo as $k => $v) {
	$q .= $k . ", ";
}
$q .= "lastPollingRun) VALUES ('POLLINFO', ";
foreach($pollingInfo as $k => $v) {
	$q .= $v . ", ";
}
$q .= "NOW()) ON DUPLICATE KEY UPDATE ";
foreach($pollingInfo as $k => $v) {
	if($k == "id") {
		continue;
	}else {
		$q .= $k . " = " . $v . ", ";
	}
}
$q .= "lastPollingRun = NOW()";

wxc_putMySql($sql_connection, $q);

$mapInfo = [];
$mapInfo['localnode'] = $localNodeName;
$mapInfo['mapTileServers'] = $USER_SETTINGS['mapTileServers'];
$mapInfo['title'] = $USER_SETTINGS['pageTitle'];
$mapInfo['attribution'] = $USER_SETTINGS['attribution'];
$mapInfo['mapContact'] = $USER_SETTINGS['mapContact'];
$mapInfo['kilometers'] = $USER_SETTINGS['kilometers'];
$mapInfo['webpageDataDir'] = "";
$mapInfo['mapCenterCoords'] = array($USER_SETTINGS['map_center_coordinates']['lat'], $USER_SETTINGS['map_center_coordinates']['lon']);
$mapInfo['mapInitialZoom'] = $USER_SETTINGS['map_initial_zoom_level'];
$mapInfo['totalNodesInDB'] = count(wxc_getMysqlFetchAll("SELECT node from node_info where lat is not null"));
$mapInfo['weekPlusOld'] = count(wxc_getMysqlFetchAll("select node from node_info where last_seen is null or last_seen < now() - interval 1 week"));

$pollingInfo['lastPollingRun'] = gmdate("Y-m-d H:i:s");

if($TEST_MODE) {
	echo "Creating webpage data files in: ". $USER_SETTINGS['webpageDataDir'] . "... ";
}
$mapDataFileName = $USER_SETTINGS['webpageDataDir'] . "/map_data.js";
$fh = fopen($mapDataFileName, "w") or die ("could not open file");
fwrite($fh, createJS($pollingInfo, $mapInfo, $versions));
fclose($fh);

$node_report_data_json = $USER_SETTINGS['webpageDataDir'] . "/node_report_data.json";

createNodeReportJSON($sql_connection, $node_report_data_json);

if($TEST_MODE) {
	echo wxc_addColor("Done!", "greenBold");
	echo "\n\n";
}

//upload the js and json file to another server via SSH
//must be able to login via SSH key with no password for this to work.
if($USER_SETTINGS['uploadToCloud']) {
	foreach($USER_SETTINGS['cloudServer'] as $k => $v) {
		if(str_contains($k, "Example")) {
			continue;
		}
		if($TEST_MODE) {
			echo "Uploading map data files to the 'cloud' via SSH to: " . $k  . "... ";
		}
		exec("scp -i " . $USER_SETTINGS['cloudSSHKeyFile'] . " " . $mapDataFileName . " " . $v . "/map_data.js");
		exec("scp -i " . $USER_SETTINGS['cloudSSHKeyFile'] . " " . $node_report_data_json . " " . $v . "/node_report_data.json");
		if($TEST_MODE) {
			echo wxc_addColor("Done!", "greenBold");
			echo "\n";
		}
	}
}
?>
