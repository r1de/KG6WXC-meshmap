<?php
function create_sqlite_db($filename) {
//	$createdNewDB = false;
//	if(!file_exists($INCLUDE_DIR . "/sqlite3_db/mesh-map.sqlite")) {
//		$createdNewDB = true;
		$sqlite3_db = new SQLite3($filename);
//		$sqlite3_db = new SQLite3($INCLUDE_DIR . "/sqlite3_db/mesh-map.sqlite");
		$node_info = <<<EOF
CREATE TABLE IF NOT EXISTS 'node_info' (
  'node' text,
  'wlan_ip' text unique,
  'last_seen' text,
  'uptime' text,
  'loadavg' text,
  'model' text,
  'firmware_version' text,
  'ssid' text,
  'channel' text,
  'chanbw' text,
  'tunnel_installed' text,
  'active_tunnel_count' text,
  'lat' real,
  'lon' real,
  'wifi_mac_address' text,
  'api_version' text,
  'board_id' text,
  'firmware_mfg' text,
  'grid_square' text,
  'lan_ip' text,
  'services' text,
  'location_fix' int,
  'link_info',
  'hopsAway' int,
  'mesh_gateway' int
);
EOF;
		$map_info = <<<EOF
	  CREATE TABLE IF NOT EXISTS 'map_info' (
	  'id' text primary key,
	  'table_or_script_name' text ,
	  'table_records_num' int,
	  'table_update_num' int,
	  'table_last_update' text,
	  'script_last_run' text,
	  'currently_running' int
	)
EOF;
		
		$marker_info = <<<EOF
	  CREATE TABLE IF NOT EXISTS 'marker_info' (
	  'id' int primary key,
	  'name' text,
	  'description' text,
	  'type' text,
	  'lat' real NOT NULL,
	  'lon' real NOT NULL
	)
EOF;
	
		$topology = <<<EOF
	  CREATE TABLE IF NOT EXISTS 'topology' (
	  'node' text,
	  'nodelat' real,
	  'nodelon' real,
	  'linkto' text,
	  'linklat' real,
	  'linklon' real,
	  'cost' real,
	  'distance' real,
	  'bearing' real,
	  'lastupd' text,
	  'band' text
	)
EOF;
	
		$users = <<<EOF
	  CREATE TABLE IF NOT EXISTS 'users' (
	  'id' int,
	  'admin' int,
	  'user' text,
	  'passwd' text,
	  'last_login'
	)
EOF;

		$aredn_info = <<<EOF
CREATE TABLE IF NOT EXISTS 'aredn_info' (
'stable_version' text,
'nightly_build_num' int,
'nightly_build_date' text
)
EOF;

	$sqlite3_db->exec($node_info);
	$sqlite3_db->exec($map_info);
	$sqlite3_db->exec($marker_info);
	$sqlite3_db->exec($topology);
	$sqlite3_db->exec($users);
	$sqlite3_db->exec($aredn_info);
//	}
//	return $createdNewDB;
return $sqlite3_db;
}
?>