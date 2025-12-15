<script setup lang="ts">
import { computed } from 'vue'
import { useData, withBase } from 'vitepress'

const { lang, site } = useData()

const isZh = computed(() => (lang.value ?? '').toLowerCase().startsWith('zh'))

const vstatsConfig = computed(() => (site.value.themeConfig as any)?.vstats ?? {})
const version = computed(() => vstatsConfig.value?.version ?? '')
const mainSiteUrl = computed(() => vstatsConfig.value?.mainSiteUrl ?? 'https://vstats.zsoft.cc/')

const changelogLink = computed(() => (isZh.value ? withBase('/zh/changelog') : withBase('/changelog')))
const mainSiteText = computed(() => (isZh.value ? '返回主站' : 'Back to Main Site'))
const changelogText = computed(() => (isZh.value ? '更新日志' : 'Changelog'))
</script>

<template>
  <div class="vstats-nav-extras">
    <a
      v-if="version"
      class="vstats-nav-chip"
      :href="changelogLink"
      :aria-label="`${changelogText} (${version})`"
      :title="`${changelogText} (${version})`"
    >
      {{ version }}
    </a>

    <a
      class="vstats-nav-button"
      :href="mainSiteUrl"
      target="_blank"
      rel="noreferrer"
      :aria-label="mainSiteText"
      :title="mainSiteText"
    >
      {{ mainSiteText }}
    </a>
  </div>
</template>


