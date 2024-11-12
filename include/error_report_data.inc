<?php
function createErrorReportJSON($errorReportDataFile) {
	$h = fopen(dirname(__FILE__) . "/../logs/polling_errors.log", "r");
	$jsonOut = fopen($errorReportDataFile, "w");
	$lineAr['data'] = array();
	while(!feof($h)) {
		$line = fgets($h);
		if(strpos($line, '#') !== false || empty(trim($line))) {
			continue;
		}else {
			$data = array();
			$line = preg_replace('#\\x1b[[][^A-Za-z]*[A-Za-z]#', '', $line);
			list($data['date'], $data['error'], $data['node'], $data['reason']) = explode(" : ", trim($line));
			$lineAr['data'][] = $data;
		}
	}
	fwrite($jsonOut, json_encode($lineAr));
	fclose($jsonOut);
}
?>
