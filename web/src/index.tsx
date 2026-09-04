import { render } from "solid-js/web"
import App from "./App"
import { loadTheme } from "./theme"
import "./style.css"

loadTheme()

render(() => <App />, document.getElementById("root")!)
