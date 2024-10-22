<?php
function outputToFile($results, $logFilePath) {
	$logFile = fopen($logFilePath, "a");
	if(is_array($results)) {
		fwrite($logFile, "Name: " . wxc_addColor($results['node'], "purple") . "\n");
		fwrite($logFile, "MAC Address: ");
		if($results['wifi_mac_address']) {
			fwrite($logFile, wxc_addColor($results['wifi_mac_address'], "green") . "\n");
		}else {
			fwrite($logFile, wxc_addColor("No MAC Address Found!", "red") . "/n");
		}
		fwrite($logFile, "Model: " . $results['model'] . "\n");
		if ($results['firmware_version'] !== $GLOBALS['USER_SETTINGS']['current_stable_fw_version']) {
			if (version_compare($results['firmware_version'], $GLOBALS['USER_SETTINGS']['current_stable_fw_version'], "<")) {
				if ($results['firmware_version'] === "Linux" || $results['firmware_version'] === "linux") {
					fwrite($logFile, "Firmware: " . $results['firmware_version'] . "  <- \033[1;32mViva Linux!!!\033[0m\n");
				}else {
					fwrite($logFile, "Firmware: " . $results['firmware_mfg'] . " ");
					fwrite($logFile, wxc_addColor($results['firmware_version'], "red"));
					fwrite($logFile, wxc_addColor(" Should update firmware!", "red") . "/n");
				}
			}
			if (version_compare($results['firmware_version'], $GLOBALS['USER_SETTINGS']['current_stable_fw_version'], ">")) {
				//echo "Firmware: " . $result['firmware_mfg'] . " " . $result['firmware_version'] . "  <- \033[31mBeta firmware!\033[0m\n";
				fwrite($logFile, "Firmware: " . $results['firmware_mfg'] . " ");
				fwrite($logFile, wxc_addColor($results['firmware_version'], "orange"));
				fwrite($logFile, wxc_addColor(" Beta firmware!", "orange") . "/n");
			}
		}else {
			//echo "Firmware Version: " . $firmware_version . "\n";
			fwrite($logFile, "Firmware: \033[32m" . $results['firmware_mfg'] . " " . $results['firmware_version'] . "\033[0m\n");
		}
		
		fwrite($logFile, "LAN ip: ");
		if ($results['lan_ip'] == "Not Available") {
			fwrite($logFile, wxc_addColor($results['lan_ip'], "orange"));
		}else {
			fwrite($logFile, $results['lan_ip']);
		}
		fwrite($logFile, " WLAN ip: " . $results['wlan_ip'] . "\n");
		
		if (($results['lat']) && ($results['lon'])) {
			fwrite($logFile, "Location: \033[32m" . $results['lat'] . ", " . $results['lon']. "\033[0m\n");
			//}elseif ($nodeLocationFixed = 1) {
			//	echo "\033[31mNo Location Info Set!\033[0m (FIXED!)\n";
			//	$nodeLocationFixed = 0;
		}else {
			fwrite($logFile, "\033[31mNo Location Info Set!\033[0m\n");
			//K6GSE's solution to deal with non-null values in the DB
			//$lat = 0.0;
			//$lon = 0.0;
			//end
		}
		
		if($results['uptime'] != "Not Available") {
			fwrite($logFile, "Uptime: " . wxc_addColor($results['uptime'], "green") . "\n");
		}else {
			fwrite($logFile, "Uptime: " . wxc_addColor($results['uptime'], "orange") . "\n");
		}
		
		fwrite($logFile, "Loads: ");
		if($results['loadavg'] != "Not Available") {
			$ldavgs = unserialize($results['loadavg']);
			$ld1 = $ldavgs[0];
			$ld5 = $ldavgs[1];
			$ld15 = $ldavgs[2];
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
			
			fwrite($logFile, "1 min: " . $ld1 . " 5 min: " . $ld5 . " 15 min: " . $ld15 . "\n");
		}else {
			fwrite($logFile, wxc_addColor($results['loadavg'], "orange") . "\n");
		}
		
		fwrite($logFile, "Mesh RF: ");
		if($results['meshRF'] == "off") {
			fwrite($logFile, wxc_addColor($results['meshRF'], "orange"));
		}else {
			fwrite($logFile, wxc_addColor($results['meshRF'], "green"));
		}
		
		
		fwrite($logFile, "  SSID: ");
		if($results['meshRF'] == "off") {
			fwrite($logFile, wxc_addColor($results['ssid'], "orange"));
		}else {
			fwrite($logFile, wxc_addColor($results['ssid'], "green"));
		}
		
		fwrite($logFile, " Channel: " . $results['channel'] . " Band: " . wxc_checkBand($results['channel'], $results['board_id']) . "\n");
		
		
		fwrite($logFile, "API Version: ");
		switch($results['api_version']) {
			case "1.11":
				fwrite($logFile, wxc_addColor($results['api_version'], "greenBold") . "\n");
				break;
			case "1.9":
				fwrite($logFile, wxc_addColor($results['api_version'], "green") . "\n");
				break;
			case "1.5":
				fwrite($logFile, wxc_addColor($results['api_version'], "orange") . "\n");
				break;
			case "1.3":
				fwrite($logFile, wxc_addColor($results['api_version'], "red") . "\n");
				break;
			default:
				fwrite($logFile, wxc_addColor($results['api_version'], "redBold") . "\n");
				break;
		}
		
		fwrite($logFile, "Tunnels: ");
		fwrite($logFile, wxc_addColor($results['tunnel_installed'], "orange"));
		fwrite($logFile, "  Count: ");
		fwrite($logFile, wxc_addColor($results['active_tunnel_count'], "orange"));
		fwrite($logFile, "\n");
		fwrite($logFile, "Hops away from localnode: " . $results['hopsAway']);
		fwrite($logFile, "\n");
		
		//links
		if(isset($results['link_info'])) {
			$linkedTo = unserialize($results['link_info']);
			fwrite($logFile, "Linked to: \n");
			foreach($linkedTo as $link) {
				if(isset($link['hostname'])) {
					fwrite($logFile, "  " . $link['hostname'] . " |" . " Type: ");
				}else {
					fwrite($logFile, "  " . $link['lastHopIP'] . " |" . " Type: ");
				}
				if(isset($link['linkType'])) {
					switch($link['linkType']) {
						case "RF":
							fwrite($logFile, wxc_addColor($link['linkType'], "greenBold"));
							break;
						case "DTD":
							fwrite($logFile, wxc_addColor($link['linkType'], "green"));
							break;
						case "TUN":
							fwrite($logFile, wxc_addColor($link['linkType'], "orange"));
							break;
					}
				}else {
					fwrite($logFile, wxc_addColor("UNKNOWN", "redBold"));
				}
				if(isset($link['olsrInterface'])) {
					fwrite($logFile, " | Interface: " . $link['olsrInterface']);
				}else {
					fwrite($logFile, " | Interface: " . wxc_addColor("UNKNOWN", "redBold"));
				}
				if(isset($link['tx_rate'])) {
					fwrite($logFile, " | TX: ");
					if($link['tx_rate'] >= "60") {
						fwrite($logFile, wxc_addColor($link['tx_rate'], "greenBold"));
					}elseif($link['tx_rate'] < 60 && $link['tx_rate'] >= 35) {
						fwrite($logFile, wxc_addColor($link['tx_rate'], "green"));
					}elseif($link['tx_rate'] < 35 && $link['tx_rate'] >= 20) {
						fwrite($logFile, wxc_addColor($link['tx_rate'], "orange"));
					}elseif($link['tx_rate'] < 20) {
						fwrite($logFile, wxc_addColor($link['tx_rate'], "red"));
					}elseif($link['tx_rate'] < 10) {
						fwrite($logFile, wxc_addColor($link['tx_rate'], "redBold"));
					}
					fwrite($logFile, "Mbps");
				}
				if(isset($link['rx_rate'])) {
					fwrite($logFile, " | RX: ");
					if($link['rx_rate'] >= "60") {
						fwrite($logFile, wxc_addColor($link['rx_rate'], "greenBold"));
					}elseif($link['rx_rate'] < 60 && $link['rx_rate'] >= 35) {
						fwrite($logFile, wxc_addColor($link['rx_rate'], "green"));
					}elseif($link['rx_rate'] < 35 && $link['rx_rate'] >= 20) {
						fwrite($logFile, wxc_addColor($link['rx_rate'], "orange"));
					}elseif($link['rx_rate'] < 20) {
						fwrite($logFile, wxc_addColor($link['rx_rate'], "red"));
					}elseif($link['rx_rate'] < 10) {
						fwrite($logFile, wxc_addColor($link['rx_rate'], "redBold"));
					}
					fwrite($logFile, "Mbps");
					//. $link['rx_rate'];
				}
				if(isset($link['signal']) && isset($link['noise'])) {
					fwrite($logFile, " | S/N: " . $link['signal'] . "/" . wxc_addColor($link['noise'], "red"));
				}
				//echo "\n";
				if(isset($link['pathCost'])) {
					fwrite($logFile, "\n    Path Cost: ". $link['pathCost']);
				}
				if(isset($link['ansn'])) {
					fwrite($logFile, " | ansn: " . $link['ansn']);
				}
				if(isset($link['ansnEdge'])) {
					fwrite($logFile, " | ansn edge: " . $link['ansnEdge']);
				}
				if(isset($link['validityTime'])) {
					fwrite($logFile, " | Validity Time: " . $link['validityTime']);
				}
				if(isset($link['refCount'])) {
					fwrite($logFile, " | RefCount: " . $link['refCount']);
				}
				if(isset($link['msgSeq'])) {
					fwrite($logFile, " | Msg Seq: " . $link['msgSeq']);
				}
				if(isset($link['errSeq'])) {
					fwrite($logFile, " | Err Seq: " . $link['errSeq']);
				}
				if(isset($link['errSeqValid'])) {
					fwrite($logFile, " | Err Seq Valid: ");
					fwrite($logFile, $link['errSeqValid'] ? 'true' : 'false');
				}
				if(isset($link['tcIgnored'])) {
					fwrite($logFile, " | Tc Ignored: " . $link['tcIgnored']);
				}
				if(isset($link['tcEdgeCost'])) {
					fwrite($logFile, " | Tc Edge Cost: " . $link['tcEdgeCost']);
				}
				fwrite($logFile, "\n");
			}
		}
	}
	fwrite($logFile, "\n");
	fclose($logFile);
} //<- end output to file
?>
