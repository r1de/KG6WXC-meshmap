<?php
//sql functions
//main sql connection
function wxc_connectToMySQL() {
	$GLOBALS['sql_connection'] = mysqli_connect($GLOBALS['USER_SETTINGS']['sql_server'],$GLOBALS['USER_SETTINGS']['sql_user'],$GLOBALS['USER_SETTINGS']['sql_passwd'],$GLOBALS['USER_SETTINGS']['sql_db']) or die('Could not connect to mySQL database: ' . mysqli_error($GLOBALS['connection']) . "\n");
	return $GLOBALS['sql_connection'];
}


//get things from the MySQL DB.
function wxc_getMySql ($query) {
	
	
	if (is_null($query)) {
		echo "No query sent to \"fetchMySql\" function\n";
		return 0;
	}else {
		$theQuery = mysqli_query($GLOBALS['sql_connection'], $query) or die ("Could not fetch from SQL server. " . mysqli_error($GLOBALS['sql_connection']));
		//just return the "fetch_array" it's usually what we want anyways.
		return mysqli_fetch_array($theQuery, MYSQLI_ASSOC);
		
	}
}

//put things into the MySQL DB.
function wxc_putMySql ($sql_connection, $query) {
	//	$connection = mysqli_connect($sql_server,$sql_user,$sql_passwd,$sql_db) or die('Could not connect to mySQL database: ' . mysqli_error() . "\n");
	if (is_null($query)) {
		echo "No query sent to \"wxc_putMySql\" function\n";
		return 0;
	}else {
		if(!mysqli_query($sql_connection, $query)) {
			die ("Could not send to SQL.\n" . "Query: " . $query . "\n" . mysqli_error($GLOBALS['sql_connection']));
			return 0;
		}
		return 1;
	}
}

function wxc_getMysqlFetchAll ($query) {
	//$results = array();
	if (empty($query)) {
		return 0;
	}else {
		$theQuery = mysqli_query($GLOBALS['sql_connection'], $query) or die ("Could not fetch from SQL server. " . mysqli_error($GLOBALS['sql_connection']));
		//foreach $item in $query as $whatever
		return mysqli_fetch_all($theQuery, MYSQLI_ASSOC);
		//$return 1;
	}
}

?>