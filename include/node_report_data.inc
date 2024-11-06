<?php
function createNodeReportJSON($sql_connection, $nodeReportDataFile) {
	$sql_connection = wxc_connectToMySQL();
	$node_info_result = mysqli_query($sql_connection, "SELECT *, concat(lat, ', ', lon) as location FROM node_info where node is not null") or die("Could not run node info query.");

	$allDevices = array();

	if(mysqli_num_rows($node_info_result)) {
		while($row = mysqli_fetch_array($node_info_result, MYSQLI_ASSOC)) {
			$allDevices['data'][] = $row;
		}
	}
	$nrfh = fopen($nodeReportDataFile, "w");
	fwrite($nrfh, json_encode($allDevices));
	fclose($nrfh);
}
?>
