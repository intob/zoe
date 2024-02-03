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
  fetch("https://lstn.swissinfo.ch", {
    method: "POST",
    headers: {
      "X_TYPE": "LOAD",
      "X_USR": localStorage.usr,
      "X_SESS": sessionStorage.sess,
      "X_CID": randCidForNow()
    }
  })
})

window.addEventListener("beforeunload", async() => {
  // make request
})
