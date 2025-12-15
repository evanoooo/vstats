<script setup lang="ts">
import { computed } from 'vue'
import { useData, withBase } from 'vitepress'

type Product = 'server' | 'cloud' | 'cli'

const { lang, page } = useData()

const product = computed<Product>(() => {
  // Use page.relativePath so SSR pre-render picks correct icon per page.
  const p = page.value.relativePath || ''
  if (p === 'cloud/index.md' || p.startsWith('cloud/')) return 'cloud'
  if (p === 'cli/index.md' || p.startsWith('cli/')) return 'cli'
  if (p === 'zh/cloud/index.md' || p.startsWith('zh/cloud/')) return 'cloud'
  if (p === 'zh/cli/index.md' || p.startsWith('zh/cli/')) return 'cli'
  return 'server'
})

const src = computed(() => {
  switch (product.value) {
    case 'cloud':
      return withBase('/logo-cloud.svg')
    case 'cli':
      return withBase('/logo-cli.svg')
    default:
      return withBase('/logo-server.svg')
  }
})

const alt = computed(() => {
  const isZh = (lang.value ?? '').toLowerCase().startsWith('zh')
  switch (product.value) {
    case 'cloud':
      return isZh ? 'vStats Cloud 文档' : 'vStats Cloud Docs'
    case 'cli':
      return isZh ? 'vStats CLI 文档' : 'vStats CLI Docs'
    default:
      return isZh ? 'vStats 自部署文档' : 'vStats Server Docs'
  }
})
</script>

<template>
  <img class="vstats-product-logo" :src="src" :alt="alt" />
</template>


