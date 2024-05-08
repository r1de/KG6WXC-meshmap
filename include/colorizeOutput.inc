<?php
function wxc_echoWithColor ($text, $color) {
	switch ($color) {
		case "alert":
			echo "\033[41m\033[30m\033[1m" . $text . "\033[0m";
			break;
		case "bold":
			echo "\033[1m" . $text . "\033[0m";
			break;
		case "grey":
			echo "\033[35m" . $text . "\033[0m";
			break;
		case "purple":
			echo "\033[35m" . $text . "\033[0m";
			break;
		case "orange":
			echo "\033[33m" . $text . "\033[0m";
			break;
		case "red":
			echo "\033[31m" . $text . "\033[0m";
			break;
		case "redBold":
			echo "\033[1m\33[31m" . $text . "\033[0m";
			break;
		case "green":
			echo "\033[32m" . $text . "\033[0m";
			break;
		case "greenBold":
			echo "\033[1m\33[32m" . $text . "\033[0m";
			break;
	}
//	if ($color == "red") {
//		echo "\033[31m" . $text . "\033[0m";
//	}
//	if ($color == 'redBold') {
//		echo "\033[1m\33[31m" . $text . "\033[0m";
//	}
//	if ($color == "purple") {
//		echo "\033[35m" . $text . "\033[0m";
//	}
//	if ($color == "orange") {
//		echo "\033[33m" . $text . "\033[0m";
//	}
//	if ($color == "green") {
//		echo "\033[32m" . $text . "\033[0m";
//	}
//	if ($color == 'greenBold') {
//		echo "\033[1m\33[32m" . $text . "\033[0m";
//	}
//	if ($color == 'alert') {
//		echo "\033[41m\033[30m\033[1m" . $text . "\033[0m";
//	}
//	if ($color == 'bold') {
//		echo "\033[1m" . $text . "\033[0m";
//	}
//	if ($color == 'grey') {
//		echo "\033[37m" . $text . "\033[0m";
//	}
}

function wxc_addColor ($text, $color) {
	switch ($color) {
		case "alert":
			return "\033[41m\033[30m\033[1m" . $text . "\033[0m";
			break;
		case "bold":
			return "\033[1m" . $text . "\033[0m";
			break;
		case "grey":
			return "\033[35m" . $text . "\033[0m";
			break;
		case "purple":
			return "\033[35m" . $text . "\033[0m";
			break;
		case "orange":
			return "\033[33m" . $text . "\033[0m";
			break;
		case "red":
			return "\033[31m" . $text . "\033[0m";
			break;
		case "redBold":
			return "\033[1m\33[31m" . $text . "\033[0m";
			break;
		case "green":
			return "\033[32m" . $text . "\033[0m";
			break;
		case "greenBold":
			return "\033[1m\33[32m" . $text . "\033[0m";
			break;
	}
//	if ($color == "red") {
//		return "\033[31m" . $text . "\033[0m";
//	}
//	if ($color == "purple") {
//		return "\033[35m" . $text . "\033[0m";
//	}
//	if ($color == "orange") {
//		return "\033[33m" . $text . "\033[0m";
//	}
//	if ($color == "green") {
//		return "\033[32m" . $text . "\033[0m";
//	}
//	if ($color == 'greenBold') {
//		echo "\033[1m\33[32m" . $text . "\033[0m";
//	}
//	if ($color == 'redBold') {
//		return "\033[1m\33[31m" . $text . "\033[0m";
//	}
//	if ($color == 'alert') {
//		return "\033[41m\033[30m\033[1m" . $text . "\033[0m";
//	}
//	if ($color == 'bold') {
//		return "\033[1m" . $text . "\033[0m";
//	}
//	if ($color == 'grey') {
//		return "\033[35m" . $text . "\033[0m";
//	}
}
?>