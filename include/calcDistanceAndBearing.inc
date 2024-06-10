<?php
function wxc_getDistanceAndBearing (float $node_lat, float $node_lon, float $linked_node_lat, float $linked_node_lon) {
	//kept in case we need it again
	//this is now done in the mysql server
	//the bearing and distance fields contain a math expression that does the same thing.
	//the distance expression is:
	// round(2*asin(sqrt(pow(sin((radians(linklat)-radians(nodelat))/2),2)+cos(radians(nodelat))*cos(radians(linklat))*pow(sin((radians(linklon)-radians(nodelon))/2),2)))*3959,2)
	//the bearing expression is:
	// round(mod(degrees(atan2(sin(radians(linklon)-radians(nodelon))*cos(radians(linklat)), cos(radians(nodelat))*sin(radians(linklat))-sin(radians(nodelat))*cos(radians(linklat))*cos(radians(linklon)-radians(nodelon)))) + 360,360),1)
	//or I guess you could use this function, but I'm not even sure it works
	
	//That can be done in later MySQL version (and in MariaDB 10) but now all the distance and bearing
	//is calculated via a trigger on the topology table in the database.
	//it was changed this way to have it be more compatible with other versions of MySQL
	
	//Find the distance and bearing between the 2 nodes
	$earthRadiusKM		= 6371;
	$earthRadiusMiles	= 3959;
	$distanceLatFrom 	= deg2rad($node_lat);
	$distanceLonFrom 	= deg2rad($node_lon);
	$distanceLatTo		= deg2rad($linked_node_lat);
	$distanceLonTo		= deg2rad($linked_node_lon);
	
	$latDelta			= $distanceLatTo - $distanceLatFrom;
	$lonDelta			= $distanceLonTo - $distanceLonFrom;
	
	$angleBetweenTheTwoPoints = 2 * asin(sqrt(pow(sin($latDelta / 2), 2) + cos($distanceLatFrom) * cos($distanceLatTo) * pow(sin($lonDelta / 2), 2)));
	
	$distanceKM = $angleBetweenTheTwoPoints * $earthRadiusKM;
	$distanceKM = round($distanceKM, 2);
	$distanceMiles = $angleBetweenTheTwoPoints * $earthRadiusMiles;
	$distanceMiles = round($distanceMiles, 2);
	
	$bearing = (rad2deg(atan2(sin(deg2rad($linked_node_lon) - deg2rad($node_lon)) * cos(deg2rad($linked_node_lat)), cos(deg2rad($node_lat)) * sin(deg2rad($linked_node_lat)) - sin(deg2rad($node_lat)) * cos(deg2rad($linked_node_lat)) * cos(deg2rad($linked_node_lon) - deg2rad($node_lon)))) + 360) % 360;
	
	return array("distanceKM" => $distanceKM, "distanceMiles" => $distanceMiles, "bearing" => $bearing);
}
?>
