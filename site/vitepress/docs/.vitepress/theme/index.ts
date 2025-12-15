import DefaultTheme from 'vitepress/theme'
import { h } from 'vue'
import NavBarExtras from './components/NavBarExtras.vue'
import ProductLogo from './components/ProductLogo.vue'
import './custom.css'

export default {
  ...DefaultTheme,
  Layout() {
    return h(DefaultTheme.Layout, null, {
      'nav-bar-title-before': () => h(ProductLogo),
      'nav-bar-content-after': () => h(NavBarExtras),
    })
  },
}

