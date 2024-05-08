<?php
function check_aredn_stable_version() {
	if(function_exists('curl_version')) {
		$ch = curl_init();
		curl_setopt($ch, CURLOPT_CUSTOMREQUEST, "GET");
		curl_setopt($ch, CURLOPT_RETURNTRANSFER, 1);
		
		//AREDN STABLE VERSION NUMBER
		curl_setopt($ch, CURLOPT_URL, "http://downloads.arednmesh.org/releases/");
		$AREDN_STABLE_PAGE = curl_exec($ch);
		preg_match_all('/\/">(.*)\/<\/a/', $AREDN_STABLE_PAGE, $versions);
		array_pop($versions[1]);
		natsort($versions[1]);
		$VERSION = end($versions[1]);
		unset($AREDN_STABLE_PAGE);
		unset($versions);

		return $VERSION;
	}
}
function check_aredn_nightly_build_number() {
	if(function_exists('curl_version')) {
		$ch = curl_init();
		curl_setopt($ch, CURLOPT_CUSTOMREQUEST, "GET");
		curl_setopt($ch, CURLOPT_RETURNTRANSFER, 1);
		
		//AREDN NIGHTLY BUILD NUMBER (AND DATE)
		curl_setopt($ch, CURLOPT_URL, "http://downloads.arednmesh.org/snapshots/targets/ath79/generic/");
		$AREDN_NIGHTLY_PAGE = curl_exec($ch);
		$VER_START = stripos($AREDN_NIGHTLY_PAGE, '<a href="aredn-');
		$VER_END = stripos($AREDN_NIGHTLY_PAGE, '</tr>', $VER_START);
		$NIGHTLY_VERSION = substr($AREDN_NIGHTLY_PAGE , $VER_START, $VER_END - $VER_START);
		unset($AREDN_NIGHTLY_PAGE);
		$NDATE_START = stripos($NIGHTLY_VERSION, '"right"');
		$NIGHTLY_BUILD_DATE = substr($NIGHTLY_VERSION, $NDATE_START, $NDATE_START - 24);
		$VER_START = stripos($NIGHTLY_VERSION, '">');
		$VER_END = stripos($NIGHTLY_VERSION, '</a>');
		$NIGHTLY_VERSION = substr($NIGHTLY_VERSION, $VER_START, $VER_END - $VER_START);
		$NIGHTLY_VERSION = substr(strrchr($NIGHTLY_VERSION, ">"), 6);
		preg_match('/-(.*?)-/', $NIGHTLY_VERSION, $nightly_build_number);
		preg_match('/>(.*?)</', $NIGHTLY_BUILD_DATE, $nightly_build_date);
		unset($NIGHTLY_VERSION);
		unset($NIGHTLY_BUILD_DATE);
		unset($VER_START);
		unset($VER_END);
		unset($NDATE_START);
		$nightly_build_info['number'] = $nightly_build_number[1];
		$nightly_build_info['date'] = $nightly_build_date[1];
		unset($nightly_build_number);
		unset($nightly_build_date);

		return $nightly_build_info;
	}
}
?>
