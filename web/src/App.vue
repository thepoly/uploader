<template>
  <div id="app">
    <nav class="navbar is-danger" role="navigation">
      <div class="navbar-brand">
        <h1 class="navbar-item title">Poly uploader</h1>
      </div>
    </nav>
    <div class="container">
      <router-view/>
    </div>
  </div>
</template>

<script>
export default {
  name: 'app'
}
</script>

<style>
@import '../node_modules/bulma/css/bulma.css';
@import '../node_modules/medium-editor/dist/css/medium-editor.min.css';
@import '../node_modules/medium-editor/dist/css/themes/beagle.min.css';
</style>
