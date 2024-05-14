<?php
//output info to console in test mode(s)
function outputToConsoleNew($info) {
	if($GLOBALS['testNodePolling']) {
		if(is_array($info)) {
			echo "Name: " . wxc_AddColor($info['node'], "purple") . "\n";
			
		}
	}
}
function outputToConsole($results) {
//	if ($GLOBALS['testNodePolling']) {
		if(is_array($results)) {
			echo "Name: "; wxc_echoWithColor($results['node'], "purple"); echo "\n";
			echo "MAC Address: ";
			if($results['wifi_mac_address']) {
				wxc_echoWithColor($results['wifi_mac_address'], "green");
				echo "\n";
			}else {
				wxc_echoWithColor("No MAC Address Found!", "red");
			}
			echo "Model: " . $results['model'] . "\n";
			if ($results['firmware_version'] !== $GLOBALS['USER_SETTINGS']['current_stable_fw_version']) {
				if (version_compare($results['firmware_version'], $GLOBALS['USER_SETTINGS']['current_stable_fw_version'], "<")) {
					if ($results['firmware_version'] === "Linux" || $results['firmware_version'] === "linux") {
						echo "Firmware: " . $results['firmware_version'] . "  <- \033[1;32mViva Linux!!!\033[0m\n";
					}else {
						echo "Firmware: " . $results['firmware_mfg'] . " ";
						wxc_echoWithColor($results['firmware_version'], "red");
						wxc_echoWithColor(" Should update firmware!", "red");
						echo "\n";
					}
				}
				if (version_compare($results['firmware_version'], $GLOBALS['USER_SETTINGS']['current_stable_fw_version'], ">")) {
					//echo "Firmware: " . $result['firmware_mfg'] . " " . $result['firmware_version'] . "  <- \033[31mBeta firmware!\033[0m\n";
					echo "Firmware: " . $results['firmware_mfg'] . " ";
					wxc_echoWithColor($results['firmware_version'], "orange");
					wxc_echoWithColor(" Beta firmware!", "orange");
					echo "\n";
				}
			}else {
				//echo "Firmware Version: " . $firmware_version . "\n";
				echo "Firmware: \033[32m" . $results['firmware_mfg'] . " " . $results['firmware_version'] . "\033[0m\n";
			}
			
			echo "LAN ip: ";
			if ($results['lan_ip'] == "Not Available") {
				wxc_echoWithColor($results['lan_ip'], "orange");
			}else {
				echo $results['lan_ip'];
			}
			echo " WLAN ip: " . $results['wlan_ip'] . "\n";
			
			if (($results['lat']) && ($results['lon'])) {
				echo "Location: \033[32m" . $results['lat'] . ", " . $results['lon']. "\033[0m\n";
				//}elseif ($nodeLocationFixed = 1) {
				//	echo "\033[31mNo Location Info Set!\033[0m (FIXED!)\n";
				//	$nodeLocationFixed = 0;
			}else {
				echo "\033[31mNo Location Info Set!\033[0m\n";
				//K6GSE's solution to deal with non-null values in the DB
				//$lat = 0.0;
				//$lon = 0.0;
				//end
			}
			
			if($results['uptime'] != "Not Available") {
				echo "Uptime: ";
				wxc_echoWithColor($results['uptime'], "green");
				echo "\n";
			}else {
				echo "Uptime: ";
				wxc_echoWithColor($results['uptime'], "orange");
				echo "\n";
			}
			
			echo "Loads: ";
			if($results['loadavg'] != "Not Available") {
				$ldavgs = unserialize($results['loadavg']);
				$ld1 = $ldavgs[0];
				if(isset($ldavgs[1])) { 
					$ld5 = $ldavgs[1];
				}else {
					$ld5 = "na";
				}
				if(isset($ldavgs[2])) {
					$ld15 = $ldavgs[2];
				}else {
					$ld15 = "na";
				}
				if($ld1 > 1) {
					$ld1 = wxc_addColor($ld1, "red");
				}elseif ($ld1 < 1 && $ld1 > 0.5) {
					$ld1 = wxc_addColor($ld1, "orange");
				}else {
					$ld1 = wxc_addColor($ld1, "green");
				}
				if($ld5 > 1) {
					$ld5 = wxc_addColor($ld5, "red");
				}elseif ($ld5 < 1 && $ld5 > 0.5) {
					$ld5 = wxc_addColor($ld5, "orange");
				}else {
					$ld5 = wxc_addColor($ld5, "green");
				}
				if($ld15 > 1) {
					$ld15 = wxc_addColor($ld15, "red");
				}elseif ($ld15 < 1 && $ld15 > 0.5) {
					$ld15 = wxc_addColor($ld15, "orange");
				}else {
					$ld15 = wxc_addColor($ld15, "green");
				}
				
				echo "1 min: " . $ld1 . " 5 min: " . $ld5 . " 15 min: " . $ld15 . "\n";
			}else {
				wxc_echoWithColor($results['loadavg'], "orange");
				echo "\n";
			}
			
			echo "Mesh RF: ";
			if($results['meshRF'] == "off") {
				wxc_echoWithColor($results['meshRF'], "orange");
			}else {
				wxc_echoWithColor($results['meshRF'], "green");
			}
			
			
			echo "  SSID: ";
			if($results['meshRF'] == "off") {
				wxc_echoWithColor($results['ssid'], "orange");
			}else {
				wxc_echoWithColor($results['ssid'], "green");
			}
			
			echo " Channel: " . $results['channel'] . " Band: " . wxc_checkBand($results['channel'], $results['board_id']);
			
			echo "\n";
			
			echo "API Version: ";
			switch($results['api_version']) {
				case "1.11":
					wxc_echoWithColor($results['api_version'], "greenBold");
					break;
				case "1.9":
					wxc_echoWithColor($results['api_version'], "green");
					break;
				case "1.5":
					wxc_echoWithColor($results['api_version'], "orange");
					break;
				case "1.3":
					wxc_echoWithColor($results['api_version'], "red");
					break;
				default:
					wxc_echoWithColor($results['api_version'], "redBold");
					break;
			}
			echo "\n";
			
			echo "Tunnels: ";
			wxc_echoWithColor($results['tunnel_installed'], "orange");
			echo "  Count: ";
			wxc_echoWithColor($results['active_tunnel_count'], "orange");
			echo "\n";
			echo "Hops away from localnode: " . $results['hopsAway'];
			echo "\n";
			
			//links
			if(isset($results['link_info'])) {
				$linkedTo = unserialize($results['link_info']);
				echo "Linked to: \n";
				foreach($linkedTo as $link) {
					if(isset($link['hostname'])) {
						echo "  " . $link['hostname'] . " |" . " Type: ";
					}else {
						echo "  " . $link['lastHopIP'] . " |" . " Type: ";
					}
					if(isset($link['linkType'])) {
						switch($link['linkType']) {
							case "RF":
								wxc_echoWithColor($link['linkType'], "greenBold");
								break;
							case "DTD":
								wxc_echoWithColor($link['linkType'], "green");
								break;
							case "TUN":
								wxc_echoWithColor($link['linkType'], "orange");
								break;
						}
					}else {
						wxc_echoWithColor("UNKNOWN", "redBold");
					}
					if(isset($link['olsrInterface'])) {
						echo " | Interface: " . $link['olsrInterface'];
					}else {
						echo " | Interface: " . wxc_addColor("UNKNOWN", "redBold");
					}
					if(isset($link['tx_rate'])) {
						echo " | TX: ";
						if($link['tx_rate'] >= "60") {
							wxc_echoWithColor($link['tx_rate'], "greenBold");
						}elseif($link['tx_rate'] < 60 && $link['tx_rate'] >= 35) {
							wxc_echoWithColor($link['tx_rate'], "green");
						}elseif($link['tx_rate'] < 35 && $link['tx_rate'] >= 20) {
							wxc_echoWithColor($link['tx_rate'], "orange");
						}elseif($link['tx_rate'] < 20) {
							wxc_echoWithColor($link['tx_rate'], "red");
						}elseif($link['tx_rate'] < 10) {
							wxc_echoWithColor($link['tx_rate'], "redBold");
						}
						echo "Mbps";
					}
					if(isset($link['rx_rate'])) {
						echo " | RX: ";
						if($link['rx_rate'] >= "60") {
							wxc_echoWithColor($link['rx_rate'], "greenBold");
						}elseif($link['rx_rate'] < 60 && $link['rx_rate'] >= 35) {
							wxc_echoWithColor($link['rx_rate'], "green");
						}elseif($link['rx_rate'] < 35 && $link['rx_rate'] >= 20) {
							wxc_echoWithColor($link['rx_rate'], "orange");
						}elseif($link['rx_rate'] < 20) {
							wxc_echoWithColor($link['rx_rate'], "red");
						}elseif($link['rx_rate'] < 10) {
							wxc_echoWithColor($link['rx_rate'], "redBold");
						}
						echo "Mbps";
						//. $link['rx_rate'];
					}
					if(isset($link['signal']) && isset($link['noise'])) {
						echo " | S/N: " . $link['signal'] . "/";
						wxc_echoWithColor($link['noise'], "red");
					}
					//echo "\n";
					if(isset($link['pathCost'])) {
						echo "\n    Path Cost: ". $link['pathCost'];
					}
					if(isset($link['ansn'])) {
						echo " | ansn: " . $link['ansn'];
					}
					if(isset($link['ansnEdge'])) {
						echo " | ansn edge: " . $link['ansnEdge'];
					}
					if(isset($link['validityTime'])) {
						echo " | Validity Time: " . $link['validityTime'];
					}
					if(isset($link['refCount'])) {
						echo " | RefCount: " . $link['refCount'];
					}
					if(isset($link['msgSeq'])) {
						echo " | Msg Seq: " . $link['msgSeq'];
					}
					if(isset($link['errSeq'])) {
						echo " | Err Seq: " . $link['errSeq'];
					}
					if(isset($link['errSeqValid'])) {
						echo " | Err Seq Valid: ";
						echo $link['errSeqValid'] ? 'true' : 'false';
					}
					if(isset($link['tcIgnored'])) {
						echo " | Tc Ignored: " . $link['tcIgnored'];
					}
					if(isset($link['tcEdgeCost'])) {
						echo " | Tc Edge Cost: " . $link['tcEdgeCost'];
					}
					echo "\n";
				}
			}
		}
			echo "\n";	
//	}
		
} //<- end test mode echo to screen
?>
