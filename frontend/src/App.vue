<script setup lang="ts">
import { useRoute } from 'vue-router';
const route = useRoute();
</script>

<template>
  <router-view :key="route.fullPath" />
</template>

<style>
/* Global styles in style.css */
</style>
