function randId() {
  return Math.floor(Math.random() * Math.pow(2, 32))
}

function randCidForNow() {
  return Math.floor(Math.random() * 10)
}

window.addEventListener("DOMContentLoaded", () => {
  if (!localStorage.usr) {
    localStorage.usr = randId()
  }
  if (!sessionStorage.sess) {
    sessionStorage.sess = randId()
  }
  fetch("https://zoe.swissinfo.ch", {
    method: "POST",
    headers: {
      "TYPE": "LOAD",
      "USR": localStorage.usr,
      "SESS": sessionStorage.sess,
      "CID": randCidForNow()
    }
  })
})

window.addEventListener("beforeunload", async() => {
  // make request
})
