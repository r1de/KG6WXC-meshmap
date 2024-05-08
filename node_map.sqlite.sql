-- Dumping database structure for node_map
-- CREATE DATABASE IF NOT EXISTS 'node_map';
-- USE 'node_map';

-- Data exporting was unselected.
-- Dumping structure for table node_map.map_info
CREATE TABLE IF NOT EXISTS 'map_info' (
  'id' text primary key,
  'table_or_script_name' text ,
  'table_records_num' int,
  'table_update_num' int,
  'table_last_update' text,
  'script_last_run' text,
  'currently_running' int
);

-- Data exporting was unselected.
-- Dumping structure for table node_map.marker_info
CREATE TABLE IF NOT EXISTS 'marker_info' (
  'id' int primary key,
  'name' text,
  'description' text,
  'type' text,
  'lat' real NOT NULL,
  'lon' real NOT NULL
);

-- Data exporting was unselected.
-- Dumping structure for table node_map.node_info
CREATE TABLE IF NOT EXISTS 'node_info' (
  'node' text,
  'wlan_ip' text,
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
  'hopsAway' int
);

-- Data exporting was unselected.
-- Dumping structure for table node_map.topology
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
);

-- Data exporting was unselected.
-- Dumping structure for table node_map.users
CREATE TABLE IF NOT EXISTS 'users' (
  'id' int,
  'admin' int,
  'user' text,
  'passwd' text,
  'last_login'
);
