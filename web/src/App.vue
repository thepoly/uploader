<template>
  <div id="app">
    <nav class="level is-mobile is-marginless">
      <div class="level-left">
        <div class="level-item">
          <router-link to="/" class="title has-text-white">Uploader</router-link>
        </div>
      </div>
      <div class="level-right">
        <div class="level-item">
          <img class="" src="../static/masthead_nobg.svg" width="150px" />
        </div>
      </div>
    </nav>
    <div class="container">
      <router-view/>
    </div>
    <footer class="footer">
      <div class="container">
        <div class="content has-text-centered">
          <p>Thanks for using the <strong><i>Poly</i> Uploader</strong>. Check it out <a href="https://github.com/thepoly/uploader">on GitHub</a>.</p>
        </div>
      </div>
    </footer>
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

<style scoped>
nav {
  background: #DA1E05;
  padding: 10px;
}
</style>
