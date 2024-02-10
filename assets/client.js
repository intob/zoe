function randId() {
  return Math.floor(Math.random() * Math.pow(2, 32))
}

function randCidForNow() {
  return Math.floor(Math.random() * 10)
}

// TODO: get the current article id
const cid = randCidForNow()

// set user and session id
if (!localStorage.usr) {
  localStorage.usr = randId()
}
if (!sessionStorage.sess) {
  sessionStorage.sess = randId()
}

// send time every 5 seconds
setInterval(async () => {
  // make request
  await fetch("https://zoe.swissinfo.ch", {
    method: "POST",
    headers: {
      "TYPE": "TIME",
      "USR": localStorage.usr,
      "SESS": sessionStorage.sess,
      "CID": cid,
    }
  })
}, 5000)

// send scroll info on unload
window.addEventListener("beforeunload", async() => {
  // make request
  await fetch("https://zoe.swissinfo.ch", {
    method: "POST",
    headers: {
      "TYPE": "UNLOAD",
      "USR": localStorage.usr,
      "SESS": sessionStorage.sess,
      "CID": cid,
    }
  })
})

fetch("https://zoe.swissinfo.ch", {
  method: "POST",
  headers: {
    "TYPE": "LOAD",
    "USR": localStorage.usr,
    "SESS": sessionStorage.sess,
    "CID": cid,
  }
})
