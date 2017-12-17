<template>
  <div>
    <section class="section">
      <div class="columns is-mobile">
        <div class="column">
          <h1 class="title">Stories</h1>
          <h2 class="subtitle">Select a story to start editing.</h2>
        </div>
        <div class="column">
          <div class="field is-grouped is-pulled-right">
            <p class="control">
              <a class="button" v-on:click="refresh">
                <span class="icon"><font-awesome-icon :icon="syncIcon" /></span>
                <span> </span>Refresh
              </a>
            </p>
          </div>
        </div>
      </div>
      <div class="columns is-multiline">
        <div class="column is-6" v-for="p in posts">
          <div class="card">
            <div class="card-content">
              <p class="has-text-danger is-uppercase">{{ p.kicker }}</p>
              <p class="title is-5">{{ p.headline || p.snippet.name }}</p>
              <p class="subtitle is-6">{{ p.authorName }}</p>
            </div>
            <footer class="card-footer">
              <p class="card-footer-item">{{ p.snippet.lastModified | moment("from", "now") }}</p>
              <a class="card-footer-item" v-on:click="editStory(p)">Edit</a>
            </footer>
          </div>
        </div>
      </div>
      <p v-if="posts.length === 0">No posts.</p>
    </section>
    <section class="section">
      <h1 class="title">Upload</h1>
      <h2 class="subtitle">Choose an InDesign snippet to upload. Coming soon.</h2>
      <div class="file">
        <label class="file-label">
          <input class="file-input" type="file" name="snippet">
          <span class="file-cta">
            <span class="file-icon">
              <font-awesome-icon :icon="fileIcon" />
            </span>
            <span class="file-label">
              Choose a file...
            </span>
          </span>
        </label>
      </div>
    </section>
  </div>
</template>

<script>
import FontAwesomeIcon from '@fortawesome/vue-fontawesome'
import { faFile, faSync } from '@fortawesome/fontawesome-free-solid'

import Vue from 'vue'
import VueMoment from 'vue-moment'
Vue.use(VueMoment)

export default {
  name: 'StoryEditor',
  data () {
    return {
      posts: []
    }
  },
  created () {
    this.refresh()
  },
  methods: {
    editStory (story) {
      this.$router.push({
        name: 'StoryEditor',
        params: {
          story: story
        }
      })
    },
    refresh () {
      fetch('http://127.0.0.1:8000/available-stories').then(response => {
        return response.json()
      }).then(posts => {
        this.posts = posts
      })
    }
  },
  computed: {
    fileIcon () {
      return faFile
    },
    syncIcon () {
      return faSync
    }
  },
  components: {
    FontAwesomeIcon
  }
}
</script>

<style scoped>
.card {
  height: 100%;
  display: flex;
  flex-direction: column;
}
.card .card-footer {
  margin-top: auto;
}
</style>
