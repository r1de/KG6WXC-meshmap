<?php
function wxc_checkBand ($channel, $board_id = null) {
	//band channel number arrays
	
	//900Mhz
	//On 900MHz you dont get "channels", the node lets you choose the center frequency.
	//there are 4 to choose from: "(907)", "(912)", "(917)", and "(922)".
	//unfortunatly, it doesn't come across correctly in the json file
	//all I know is that "(917)" == "4" (for now) (which makes no sense at all, it should be 3 (or 2 actually))
	//		$nineHundredMhz = array('907', '912', '917', '922', '4');
	//finding 900MHz devices by board_id now (there's only 3 of them!!)
	//should have done it this way long ago.
	$bid900 = array('0xe009','0xe1b9','0xe239');
	
	//2.4GHz
	$twoGhz = array('-1', '-2', '1', '2', '3', '4', '5', '6', '7', '8', '9', '10', '11');
	
	//3GHz channel numbers and also sometimes the node shows frequency
	$threeGhz = array('76','77','78','79','80','81','82','83','84','85','86','87','88',
			'89','90','91','92','93','94','95','96','97','98','99','3380','3385','3390',
			'3395','3400','3405','3410','3415','3420','3425','3430','3435','3440','3445',
			'3450','3455','3460','3465','3470','3475','3480','3485','3490','3495');
	
	//5GHz channel numbers
	//133 and up are US channel numbers
	//info taken from "channelmaps.pm" in the AREDN code base
	$fiveGhz = array('37','40','44','48','52','56','60','64','100','104','108',
			'112','116','120','124','128','132',
			'133','134','135','136','137','138','139','140','141','142',
			'143','144','145','146','147','148','149','150','151','152',
			'153','154','155','156','157','158','159','160','161','162',
			'163','164','165','166','167','168','169','170','171','172',
			'173','174','175','176','177','178','179','180','181','182','183','184');
	
	
	//K6GSE nice switch statement!
	switch (true) {
		case (in_array($board_id, $bid900)):
			//case (in_array($channel, $nineHundredMhz)):
			return '900MHz';
			break;
			
		case (in_array($channel, $twoGhz)):
			return '2GHz';
			break;
			
		case (in_array($channel, $threeGhz)):
			return '3GHz';
			break;
			
		case (in_array($channel, $fiveGhz)):
			return '5GHz';
			break;
			
		default:
			return 'Unknown';
	}
	
}

function parseSysinfoJson($jsonString, $ip) {
	//declare empty return array
	$results = array();

	//remove any "funny" characters from the sysinfo.json string *BEFORE* it gets decoded
	//these mainly occur when people get fancy with the description field
	//use html name codes! do not use the hex codes!
	//$jsonString = preg_replace('/[\x00-\x1F\x7F-\xFF]/', '', $jsonString);
	
	//decode json
	//$jsonString = json_decode($jsonString,true);
	
	if(is_array($jsonString)) {
		//variables and defaults
		$results['meshRF'] = "on";
		$results['ethInf'] = "eth0";
		$results['wlanInf'] = "wlan0";
		
		$results['node'] = "";
		$results['wlan_ip'] = "";
		$results['uptime'] = "Not Available";
		$results['loadavg'] = "Not Available";
		$results['model'] = "Not Available";
		$results['firmware_version'] = "Not Available";
		$results['ssid'] = "None";
		$results['channel'] = "None";
		$results['chanbw'] = "None";
		$results['tunnel_installed'] = "false";
		$results['active_tunnel_count'] = "0";
		$results['lat'] = 0.0;
		$results['lon'] = 0.0;
		$results['wifi_mac_address'] = "";
		$results['api_version'] = "0.0.0";
		$results['board_id'] = "Not Available";
		$results['firmware_mfg'] = "Not Available";
		$results['grid_square'] = "Not Available";
		$results['lan_ip'] = "Not Available";
		$results['services'] = "Not Available";
		$results['description'] = "";
		$results['mesh_supernode'] = "false";
		$results['mesh_gateway'] = "false";
//		$results['signal'] = 0;
//		$results['noise'] = 0;
		$results['freq'] = "None";
//		$results['interfaces'] = "";
		$results['link_info'] = "";
		$results['hopsAway'] = 0;

		$results['antGain'] = 0;
		$results['antBeam'] = 0;
		$results['antDesc'] = "Not Available";
		$results['antBuiltin'] = "false";
		$results['hardware'] = "";
		
//		$results['hopsAway'] = $jsonString['hopsAway'];
		
		//hack for USB150 devices
		$results['eth3975'] = 0;
		
		//hack for weird supernodes
		$br0 = 0;

		//check a couple of things at first
		if(isset($jsonString['meshrf']['status']) && $jsonString['meshrf']['status'] == "off") {
			$results['meshRF'] = "off";
		}

		//start looping through the json data array
		foreach($jsonString as $key => $value) {

			//find the local services on the node
			if($key == "services_local") {
				//strip single quote from service names
				//one station (do far) needs this.
				//I'm sure we'll need to add to it in the future.
				foreach($value as $srv => $srvInfo) {
					if(str_contains($srvInfo['name'], "'")) {
						$value[$srv]['name'] = str_replace("'", '', $srvInfo['name']);
					}
				}
				$results['services'] = serialize($value); // <-- this is an array that has been serialized!!
				continue;
			}

			//serialize link_info from the polled node
			if($key == "link_info") {
				$results['link_info'] = serialize($value);
				continue;
			}
			
			if(is_array($value) && $key != "services_local") { //<- $value is an array
				
				foreach($value as $key2 => $value2) { //<- step thru the array
					
					//find node load averages
					if($key2 === "loads") {
						$results['loadavg'] = serialize($value2); // <-- this is an array that has been serialized!!
						continue;
					}
					
					if($key == "interfaces") {
						if(is_numeric($key2)) {
							switch($value2['name']) {
								case $results['wlanInf']:
									$results['wifi_mac_address'] = $value2['mac'];
									break;
								case "br-lan":
									if(isset($value2['ip'])) {
										$results['lan_ip'] = $value2['ip'];
									}
									break;
							}
							if(isset($value2['ip']) && $value2['ip'] != "none" && !$results['eth3975'] && !$br0) {
								switch($value2['name']) {
									case "br0":
										if(str_starts_with($value2['ip'], "10.")) {
										$results['wlan_ip'] = $value2['ip'];
										$br0 = 1;
										break;
										}else {
											break;
										}
									case $results['wlanInf']:
										$results['wlan_ip'] = $value2['ip'];
										break;
										//for the HaP Lite Models
									case "wlan1":
										$results['wlan_ip'] = $value2['ip'];
										break;
									case "eth1.3975":
										$results['wlan_ip'] = $value2['ip'];
										$results['eth3975'] = 1;
										break;
									case "eth0.3975":
										$results['wlan_ip'] = $value2['ip'];
										$results['eth3975'] = 1;
										break;
									case "br-nomesh":
										$results['wlan_ip'] = $value2['ip'];
										break;
								}
							}else {
								$results['wlan_ip'] = $ip;
							}
						}else { //<- interfaces array keys are *not* numeric (it's a very very old version of the json file)
							if(isset($value2['ip']) && $value2['ip'] != "none") {
								switch($key2) {
									case $results['wlanInf']:
										$results['wifi_mac_address'] = $value2['mac'];
										$results['wlan_ip'] = $value2['ip'];
										break;
								}
							}
						}
					}
					else {
						if(isset($value2) && $value2 != '') {
							switch($key2) {
								
								//meshrf values
								case "freq":
									$results['freq'] = $value2;
									break;
								case "status":
									$results['meshRF'] = $value2;
									break;
								case "chanbw":
									$results['chanbw'] = $value2;
									break;
								case "ssid":
									$results['ssid'] = $value2;
									break;
								case "channel":
									$results['channel'] = $value2;
									break;
								case "antenna":
									if($results['meshRF'] != "off") {
										if(isset($value2['gain'])) {
											$results['antGain'] = $value2['gain'];
										}
										if(isset($value2['beamwidth'])) {
											$results['antBeam'] = $value2['beamwidth'];
										}
										if(isset($value2['description'])) {
											$results['antDesc'] = $value2['description'];
										}
										if(isset($value2['builtin'])) {
											$results['antBuiltin'] = $value2['builtin'];
										}
									}
									break;
									
									//tunnel values
								case "active_tunnel_count":
									$results['active_tunnel_count'] = $value2;
									break;
								case "tunnel_installed":
									$results['tunnel_installed'] = $value2;
									break;
									
									//uptime
								case "uptime":
									$results['uptime'] = $value2;
									break;
									
									//node info
								case "description":
									$results['description'] = $value2;
									break;
								case "model":
									$results['model'] = $value2;
									break;
								case "board_id":
									$results['board_id'] = $value2;
									break;
								case "firmware_mfg":
									$results['firmware_mfg'] = $value2;
									break;
								case "firmware_version":
									$results['firmware_version'] = $value2;
									break;
								case "mesh_gateway":
									if($value2 == "1") {
										$results['mesh_gateway'] = "true";
									}
									break;
								case "mesh_supernode":
									if($value2 == "1") {
										$results['mesh_supernode'] = "true";
									}
									break;

									//there was one "funny" version 1.5 of the json API
								case "node":
									$results['node'] = $value2;
									break;
								case "lat":
									if(isset($value2) && $value2 != '' && $value2 != " ") {
										$results['lat'] = $value2;
									}
									break;
								case "lon":
									if(isset($value2) && $value2 != '' && $value2 != " ") {
										$results['lon'] = $value2;
									}
									break;
							}
						}
					} // <- end $value2 is *not* array
					unset($key2);
				} //<- end $key2/$value2 foreach loop
				
			} // <-- end $value is_array
			else {
				//these values are in the root of the json file (for now)
				if(isset($value) && $value != '' && $value != " " && !is_null($value)) {
					switch($key) {
						case "lon":
							$results['lon'] = $value;
							break;
						case "lat":
							$results['lat'] = $value;
							break;
						case "api_version":
							$results['api_version'] = $value;
							break;
						case "grid_square":
							$results['grid_square'] = $value;
							break;
						case "node":
							$results['node'] = $value;
							break;
						case "hopsAway":
							$results['hopsAway'] = $value;
							break;
							//catch old API entries
						case "ssid":
							$results['ssid'] = $value;
							break;
						case "model":
							$results['model'] = $value;
							break;
						case "board_id":
							$results['board_id'] = $value;
							break;
						case "firmware_version":
							$results['firmware_version'] = $value;
							break;
						case "firmware_mfg":
							$results['firmware_mfg'] = $value;
							break;
						case "channel":
							$results['channel'] = $value;
							break;
						case "chanbw":
							$results['chanbw'] = $value;
							break;
						case "active_tunnel_count":
							$results['active_tunnel_count'] = $value;
							break;
						case "tunnel_installed":
							$results['tunnel_installed'] = $value;
							break;
					}
				}
			} // <- end $value is *not* array
			unset($key);
		}
		if($results['board_id'] === "0x0000") {
			$results['hardware'] = findHardware($results['model']);
		}else {
			$results['hardware'] = findHardware($results['board_id']);
		}
		return $results;
	}else {
		$results =  "array not sent to polling function";
		return $results;
	}
}

function url_get_contents($url) {
	$curlHandle = curl_init();
	curl_setopt($curlHandle, CURLOPT_URL, $url);
	curl_setopt($curlHandle, CURLOPT_CONNECTTIMEOUT, 30);
	curl_setopt($curlHandle, CURLOPT_RETURNTRANSFER, true);
	$out = curl_exec($curlHandle);
	curl_close($curlHandle);
	return $out;
}

function expiredNodes($sqlConn, $expireDays) {
//testing	$query = "select wlan_ip from node_info where last_seen <= adddate(now(), interval -" . $expireDays . " hour)";
	$query = "select wlan_ip from node_info where last_seen <= adddate(now(), interval -" . $expireDays . " day)";
	$expNodes = wxc_getMysqlFetchAll($query);
	return $expNodes;
}

function parallelPolling($func, array $arr, $procs=5) {
	$chunks = array_chunk($arr, ceil((count($arr)/$procs)));
	$pid = -1;
	$children = array();
	
	foreach($chunks as $items) {
		$pid = pcntl_fork();
		if($pid === -1) {
			die("Could not fork!");
		}else if($pid === 0) {
			// in child proc
			array_walk($items, $func);
			exit(0);
		}else {
			//in parent still
			$children[] = $pid;
		}
	}
	
	//wait for the kids to finish
	foreach($children as $pid) {
		//still the parent
		pcntl_waitpid($pid,$status);
	}
}
?>
