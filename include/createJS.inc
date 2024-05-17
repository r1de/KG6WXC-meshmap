<?php
function createJS() {
	$sql_connection = wxc_connectToMySQL();

	$node_info_result = mysqli_query($sql_connection, "SELECT * FROM node_info") or die("Could not run node info query.");
	$topology_info_result = mysqli_query($sql_connection, "SELECT * FROM topology") or die("Could not run topology query.");

	$noRFDevices = array();
	$nineDevices = array();
	$twoDevices = array();
	$threeDevices = array();
	$fiveDevices = array();

	$q = mysqli_query($sql_connection, "SELECT * FROM node_info WHERE meshRF = 'off' or channel = 'none'") or die("Could not run node info query.");
	if(mysqli_num_rows($q)) {
		while($row = mysqli_fetch_array($q, MYSQLI_ASSOC)) {
			foreach($row as $k => $v) {
				if($k === "loadavg") {
					$loadavg = array();
					$loadavg = @unserialize($v);
					$row[$k] = $loadavg;
				}
				if($k === "services") {
					$services = array();
					$services = @unserialize($v);
					$row[$k] = $services;
				}
				if($k === "link_info") {
					$link_info = array();
					$link_info = @unserialize($v);
					$row[$k] = $link_info;
				}
			}
			$noRFDevices[] = $row;
		}
	}

	$q = mysqli_query($sql_connection, "SELECT * FROM node_info WHERE board_id = '0xe009' or board_id = '0xe1b9' or board_id = '0xe239'") or die("Could not run node info query.");
	if(mysqli_num_rows($q)) {
		while($row = mysqli_fetch_array($q, MYSQLI_ASSOC)) {
			foreach($row as $k => $v) {
				if($k === "loadavg") {
					$loadavg = array();
					$loadavg = @unserialize($v);
					$row[$k] = $loadavg;
				}
				if($k === "services") {
					$services = array();
					$services = @nserialize($v);
					$row[$k] = $services;
				}
				if($k === "link_info") {
					$link_info = array();
					$link_info = @unserialize($v);
					$row[$k] = $link_info;
				}
			}
			$nineDevices[] = $row;
		}
	}

	$q = mysqli_query($sql_connection, "SELECT * FROM node_info WHERE channel <= 11 and channel != 'none'") or die("Could not run node info query.");
	if(mysqli_num_rows($q)) {
		while($row = mysqli_fetch_array($q, MYSQLI_ASSOC)) {
			foreach($row as $k => $v) {
				if($k === "loadavg") {
					$loadavg = array();
					$loadavg = @unserialize($v);
					$row[$k] = $loadavg;
				}
				if($k === "services") {
					$services = array();
					$services = @unserialize($v);
					$row[$k] = $services;
				}
				if($k === "link_info") {
					$link_info = array();
					$link_info = @unserialize($v);
					$row[$k] = $link_info;
				}
			}
			$twoDevices[] = $row;
		}
	}

	$q = mysqli_query($sql_connection, "SELECT * FROM node_info WHERE (channel >= 76 and channel <= 99) or (channel >= 3000)") or die("Could not run node info query.");
	if(mysqli_num_rows($q)) {
		while($row = mysqli_fetch_array($q, MYSQLI_ASSOC)) {
			foreach($row as $k => $v) {
				if($k === "loadavg") {
					$loadavg = array();
					$loadavg = @unserialize($v);
					$row[$k] = $loadavg;
				}
				if($k === "services") {
					$services = array();
					$services = @unserialize($v);
					$row[$k] = $services;
				}
				if($k === "link_info") {
					$link_info = array();
					$link_info = @unserialize($v);
					$row[$k] = $link_info;
				}
			}
			$threeDevices[] = $row;
		}
	}

	$q = mysqli_query($sql_connection, "SELECT * FROM node_info WHERE (channel >= 37 and channel <= 64) or (channel >= 100 and channel <= 184)") or die("Could not run node info query.");
	if(mysqli_num_rows($q)) {
		while($row = mysqli_fetch_array($q, MYSQLI_ASSOC)) {
			foreach($row as $k => $v) {
				if($k === "loadavg") {
					$loadavg = array();
					$loadavg = @unserialize($v);
					$row[$k] = $loadavg;
				}
				if($k === "services") {
					$services = array();
					$services = @unserialize($v);
					$row[$k] = $services;
				}
				if($k === "link_info") {
					$link_info = array();
					$link_info = @unserialize($v);
					$row[$k] = $link_info;
				}
			}
			$fiveDevices[] = $row;
		}
	}

	$allDevices = array();
	$allDevices['noRF'] = $noRFDevices;
	$allDevices['900'] = $nineDevices;
	$allDevices['2ghz'] = $twoDevices;
	$allDevices['3ghz'] = $threeDevices;
	$allDevices['5ghz'] = $fiveDevices;

//	header("Content-Type: text/javascript");
	return "allDevices = " . json_encode($allDevices, JSON_NUMERIC_CHECK | JSON_PRETTY_PRINT) . ";";
}
