<?php
//parallel polling script
//2021 kg6wxc
//only run from the command line
if (PHP_SAPI !== 'cli') {
	$file = basename($_SERVER['PHP_SELF']);
	exit("<style>html{text-align: center;}p{display: inline;}</style>
        <br><strong>This script ($file) should only be run from the
        <p style='color: red;'>command line</p>!</strong>
        <br>exiting...");
}
$INCLUDE_DIR = dirname(__DIR__, 1);
//$INCLUDE_DIR = dirname(__FILE__, 1);
//$INCLUDE_DIR = "..";
//get required php files
require $INCLUDE_DIR . "/include/mysqlFunctions.inc";
require $INCLUDE_DIR . "/include/pollingFunctions.inc";
require $INCLUDE_DIR . "/include/colorizeOutput.inc";
require $INCLUDE_DIR . "/include/outputToConsole.inc"; //<-**outputs to log file now**
require $INCLUDE_DIR . "/include/hardware_lookup.inc";

$api1dot9 = 0;
$api1dot8 = 0;
$api1dot7 = 0;
$apiLessThan1dot7 = 0;

global $USER_SETTINGS;

//load defaults into $USER_SETTINGS
$DEFAULT_USER_SETTINGS = parse_ini_file($INCLUDE_DIR . "/settings/user-settings.ini-default");

//check for users custom settings
$USER_SETTINGS = parse_ini_file($INCLUDE_DIR . "/settings/user-settings.ini");

//merge default settings and users custom settings
$USER_SETTINGS = array_merge($DEFAULT_USER_SETTINGS, $USER_SETTINGS);

//Get current stable firmware version and stick in proper $VAR
$GLOBALS['USER_SETTINGS']['current_stable_fw_version'] = trim(fgets(fopen($INCLUDE_DIR . "/logs/latest_versions.txt", "r")));

//open the error file to write polling ERRORS to
$err_log = fopen($INCLUDE_DIR . "/logs/polling_errors.log", "a");
//. $USER_SETTINGS['errFile'], "a");

$USE_SQL = 1;

$ip = "";
$deviceInfo = array();
if(count($argv) < 2) {
	exit(date("M j G:i:s") . " - Invalid Argument, Please supply at least a node name to poll.\n");
}else {
//get the ip to poll and link_info and hopsAway for that ip
	array_shift($argv);
	$ip = $argv[0];
	$deviceInfo[$ip] = array();
//	if(!empty($argv[1]) && $argv[1] != "") {
	if(empty($argv[1])) { //<- catch the value of "0". empty() is true when the value is falsey
		//$deviceInfo[$ip] = unserialize($argv[1]);
		$USE_SQL = $argv[1];
	}
	if(!empty($argv[2]) && $argv[2] != "") {
		$deviceInfo[$ip] = unserialize($argv[2]);
	}
}

//get devices link info from DB and not from a command line arg
$sql_connection = wxc_connectToMySQL();
$deviceLinkInfo = wxc_getMySql("SELECT link_info, hopsAway from " . $GLOBALS['USER_SETTINGS']['sql_db_tbl'] . " where wlan_ip = '" . $ip . "'");
//$deviceInfo[$ip] = mysqli_query($sql_connection, "SELECT link_info from " . $GLOBALS['USER_SETTINGS']['sql_db_tbl'] . " where wlan_ip = '" . $ip . "'");
mysqli_close($sql_connection);

$deviceInfo[$ip]['link_info'] = unserialize($deviceLinkInfo['link_info']);
$deviceInfo[$ip]['hopsAway'] = $deviceLinkInfo['hopsAway'];

//poll the node
$sysinfoJson = @file_get_contents("http://" . $ip . ":8080/cgi-bin/sysinfo.json?link_info=1&services_local=1");

if($sysinfoJson == "" || is_null($sysinfoJson)) {
	if(!isset(error_get_last()['message']) || is_null(error_get_last()['message']) || error_get_last()['message'] == "") {
		$failReason = "No error, just... nothing, null, nada.";
		fwrite($err_log, (date("M j G:i:s") . " : " . wxc_addColor("sysinfo.json was not returned", "red") . " : " . $ip . " (" . gethostbyaddr($ip) . ")" .
				"Reason : " . wxc_addColor($failReason, "redBold") . "\n"));
		exit();
	}else {
		$failReason = trim(substr(strrchr(error_get_last()['message'], ":"), 1));
		//AREDN port 8080 is going away, try new way
		if($failReason === "Connection refused") {
			$sysinfoJson = "";
			$sysinfoJson = @file_get_contents("http://" . $ip . "/cgi-bin/sysinfo.json?link_info=1&services_local=1");
			if($sysinfoJson == "" || is_null($sysinfoJson)) {
				if(!isset(error_get_last()['message']) || is_null(error_get_last()['message']) || error_get_last()['message'] == "") {
					$failReason = "No error, just... nothing, null, nada.";
					fwrite($err_log, (date("M j G:i:s") . " : " . wxc_addColor("sysinfo.json was not returned", "red") . " : " . $ip . " (" . gethostbyaddr($ip) . ")" .
							" : " . wxc_addColor($failReason, "redBold") . "\n"));
					exit();
				}else {
					$failReason = trim(substr(strrchr(error_get_last()['message'], ":"), 1));
					fwrite($err_log, (date("M j G:i:s") . " : " . wxc_addColor("sysinfo.json was not returned", "red") . " : " . $ip . " (" . gethostbyaddr($ip) . ")" .
							" : " . wxc_addColor($failReason, "redBold") . "\n"));
					exit();
				}
			}
		}else {
			fwrite($err_log, (date("M j G:i:s") . " : " . wxc_addColor("sysinfo.json was not returned", "red") . " : " . $ip . " (" . gethostbyaddr($ip) . ")" .
					" : " . wxc_addColor($failReason, "redBold") . "\n"));
			exit();
		}
	}	
}else {
	//get rid of funny characters in the json string.
	//usually caused by special characters in the node descriptions
	$sysinfoJson = preg_replace('/[\x00-\x1F\x7F-\xFF]/', '', $sysinfoJson);
	
	//build up the nodes info into an array
	$sysinfoJson = json_decode($sysinfoJson,true);

	if(is_array($sysinfoJson)) {
		foreach($sysinfoJson as $k => $v) {
// 			if($k == 'api_version') {
// 				switch($v) {
// 					case '1.9':
// 						$api1dot9++;
// 						break;
// 					case '1.8':
// 						$api1dot8++;
// 						break;
// 					case '1.7':
// 						$api1dot7++;
// 						break;
// 					default:
// 						$apiLessThan1dot7++;
// 						break;
// 				}
// 			}
/*
			if (empty($v) || !isset($sysinfoJson['lat']) || !isset($sysinfoJson['lon'])
					|| $sysinfoJson['lat'] == '' || $sysinfoJson['lon'] == '') {
				$noLocCount = 0;
//				if ($k == 'lat' || $k == 'lon' && $noLocCount) {
				if ($k == 'lat' || $k == 'lon') {
					$no_loc = fopen($INCLUDE_DIR . "/logs/no_location.log", "a");
					//$USER_SETTINGS['noLocFile'], "a");
					fwrite($no_loc, (date("M j G:i:s") . " : " . wxc_addColor("no usable location info", "orange") . " : " . $ip . " (" . $sysinfoJson['node'] . ") : Location Info Not Found!\n"));
					$noLocCount++;
					//$noLocation++;
					fclose($no_loc);
				}
			}
*/
			if($k == 'link_info') {
				foreach($v as $l => $info) {
					foreach($info as $x => $y) {
						$deviceInfo[$ip][$k][$l][$x] = $y;
					}
					unset($x);
					unset($y);
				}
				unset($l);
				unset($info);
			}else {
				$deviceInfo[$ip][$k] = $v;
			}
		}
		unset($k);
		unset($v);
	} else {
		fwrite($err_log, (date("M j G:i:s") . " : " . wxc_addColor("sysinfo.json was not parsed correctly", "red") . " : " . $ip . " : JSON_ERR_NUM: " . wxc_addColor(json_last_error(), "red") . "\n"));
		exit();
	}

	unset($sysinfoJson);
	fclose($err_log);
	//get the now fixed node info and parse it all out further
	$deviceInfo = parseSysinfoJson($deviceInfo[$ip], $ip);
	
	//find no location
	//this may catch all of them now.
	if($deviceInfo['lat'] == 0 || $deviceInfo['lon'] == 0) {
		$no_loc = fopen($INCLUDE_DIR . "/logs/no_location.log", "a");
		//$USER_SETTINGS['noLocFile'], "a");
		fwrite($no_loc, (date("M j G:i:s") . " : " . wxc_addColor("no usable location info", "orange") . " : " . $ip . " (" . $deviceInfo['node'] . ") : Location Info Not Found!\n"));
		//$noLocCount++;
		//$noLocation++;
		fclose($no_loc);
	}
	
	outputToConsole($deviceInfo, $INCLUDE_DIR . "/logs/polling_output.log"); //<-**outputs to log file now**
	
	if ($USE_SQL) {
		//build up the huge SQL query for this device
		$sqlQuery = "INSERT INTO " . $GLOBALS['USER_SETTINGS']['sql_db_tbl'] . " (";
		foreach($deviceInfo as $k => $v) {
			$sqlQuery .= $k . ", ";
		}
		$sqlQuery .= "last_seen) VALUES(";
		foreach($deviceInfo as $k => $v) {
			$sqlQuery .= "'" . $v . "', ";
		}
//		$sqlQuery .= "NOW());";
		$sqlQuery .= "NOW()) ON DUPLICATE KEY UPDATE ";
		foreach($deviceInfo as $k => $v) {
			if($k == "wlan_ip") {
				continue;
			}else {
				$sqlQuery .= $k . " = '" . $v . "', ";
			}
		}
		$sqlQuery .= "last_seen = NOW()";
		
		//connect to the SQL server from the user-settings.ini file
		$sql_connection = wxc_connectToMySQL();
		
		//dump device info into the node_info table
		wxc_putMySql($sql_connection, $sqlQuery);
		
		//close SQL connection
		mysqli_close($sql_connection);
	}else {
		exit();
	}
}
?>
