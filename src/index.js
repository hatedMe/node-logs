
// const BASE_URL = "http://api.djui.cn/tongji/findnum?";
const BASE_URL = 'http://localhost:3000/tongji/findnum?'; 
const UVMARK = "uvmark";

const ceartUserIdorGetUserId = function() {
	const val = window.localStorage.getItem(UVMARK)
	if( val ) return val 
	window.localStorage.setItem(UVMARK , Math.random().toString(36).slice(2, 15) + Date.now() );
	return val
}

const config = {
	dm : window.location.host,        // host 
	uvmark : ceartUserIdorGetUserId(),
	pvid : "10001" ,		// userid
	url : window.location.pathname ,       // url 	
	arg : window.location.search ,      // url search
	ty : 0 ,
    v: "1.0.0",  // version
    scr: window.screen.width + "x" + window.screen.height,   // devices screen
    scl: window.screen.colorDepth + "-bit",   // devices colorDepth
    lg: (navigator.language || navigator.userLanguage).toLowerCase(),  // language
	tz: (new Date).getTimezoneOffset() / 60 ,  // time
	d : +new Date, // now date
	random : Math.random().toString().slice(2, 24),   // random number  
};



const serialize = obj => {
    var str = [];
    for (var p in obj) 
        if (obj.hasOwnProperty(p)) {
            str.push(encodeURIComponent(p) + "=" + encodeURIComponent(obj[p]));
        }
    return str.join("&");
}

const siteH5 = {};
siteH5.init = function () {
	
};
siteH5.pgv = function () {
    const params = serialize(config);
    const img = new Image;
	img.onload = img.onabort = img.onerror = () => {
		img.onload = img.onabort = img.onerror = null ;
	}
	img.src = BASE_URL + params;
}

siteH5.clickStat = function( evName ) {
    const params = serialize(
		Object.assign(config ,{
			dm : "click",
			url : evName
		})
	);
	const img = new Image;
	img.onload = img.onabort = img.onerror = () => {
		img.onload = img.onabort = img.onerror = null ;
	}
	img.src = BASE_URL + params;
}


window.addEventListener('unload',function(){
	const ua = this.navigator;
	if( typeof navigator.sendBeacon !== "function" ) return ;
	const params = serialize(
		Object.assign(config ,{
			d : +new Date()
		})
	);
	ua.sendBeacon( BASE_URL + params );
});


// export default siteH5;