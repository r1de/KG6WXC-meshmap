<?php
function createNodeReportJSON($sql_connection, $nodeReportDataFile) {
	$sql_connection = wxc_connectToMySQL();
	$node_info_result = mysqli_query($sql_connection, "SELECT * FROM node_info where node is not null") or die("Could not run node info query.");

	$allDevices = array();

	if(mysqli_num_rows($node_info_result)) {
		while($row = mysqli_fetch_array($node_info_result, MYSQLI_ASSOC)) {
			$allDevices['data'][] = $row;
		}
	}
//	$nodeReportDataFile = $USER_SETTINGS['webpageDataDir'] . "/node_report.json";
//	$nodeReportDataFile = $nodeReportDataFile;
	$nrfh = fopen($nodeReportDataFile, "w");
	fwrite($nrfh, json_encode($allDevices));
	fclose($nrfh);
}
//header("Content-Type: application/json");
//echo "allDevices: " . json_encode($allDevices, JSON_NUMERIC_CHECK | JSON_PRETTY_PRINT) . ";";
//echo json_encode($allDevices);
?>
