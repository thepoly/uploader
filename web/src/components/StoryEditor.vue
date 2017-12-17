<template>
  <div>
    <section class="section">
      <div class="columns is-mobile">
        <div class="column">
          <h1 class="title">Story editor</h1>
          <h2 class="subtitle">Turn a story into a WordPress post.</h2>
        </div>
        <div class="column">
          <div class="field is-grouped is-grouped-right is-grouped-multiline">
            <p class="control">
              <a class="button is-primary" v-on:click="validate">
                <span class="icon"><font-awesome-icon :icon="syncIcon" /></span>
                <span> </span>Validate
              </a>
            </p>
            <p class="control">
              <a class="button is-primary" v-bind:disabled="!canCreatePost">
                <span class="icon"><font-awesome-icon :icon="uploadIcon" /></span>
                <span> </span>Create post
              </a>
            </p>
          </div>
        </div>
      </div>
      <div class="message is-warning" v-if="validationErrors.length > 0">
        <div class="message-header">
          <p>Validation errors</p>
        </div>
        <div class="message-body">
          <ul class="validation-errors">
            <li v-for="error in validationErrors">{{ error }}</li>
          </ul>
        </div>
      </div>
      <hr>
      <medium-editor class="has-text-danger is-uppercase has-text-weight-semibold" :text="story.kicker" :options="editorOptions" v-on:edit="editKicker" />
      <medium-editor class="title" :text="story.headline" :options="editorOptions" v-on:edit="editHeadline" />
      <medium-editor class="subtitle" :text="story.subdeck" :options="editorOptions" v-on:edit="editSubdeck" />
      <medium-editor class="author-name has-text-weight-semibold" :text="story.authorName" :options="editorOptions" v-on:edit="editAuthorName" />
      <medium-editor class="author-title" :text="story.authorTitle" :options="editorOptions" v-on:edit="editAuthorTitle" />
      <medium-editor class="is-size-5" :text="story.bodyText" :options="bodyTextEditorOptions" v-on:edit="editBodyText" />
    </section>
  </div>
</template>

<script>
import editor from 'vue2-medium-editor'
import FontAwesomeIcon from '@fortawesome/vue-fontawesome'
import { faCheck, faCloudUploadAlt } from '@fortawesome/fontawesome-free-solid'

export default {
  name: 'StoryEditor',
  data () {
    return {
      editorOptions: {
        disableReturn: true,
        toolbar: {
          buttons: ['italic']
        }
      },
      bodyTextEditorOptions: {
        toolbar: {
          buttons: ['italic', 'quote']
        }
      },
      validationErrors: [],
      didValidation: false
    }
  },
  methods: {
    validate () {
      this.story.kicker = this.story.kicker.replace(/(<br>)/gim, '')
      this.story.kicker = this.story.kicker.replace(/(&nbsp;)/gim, ' ')
      this.story.headline = this.story.headline.replace(/(<br>)/gim, '')
      this.story.headline = this.story.headline.replace(/(&nbsp;)/gim, ' ')
      this.story.subdeck = this.story.subdeck.replace(/(<br>)/gim, '')
      this.story.subdeck = this.story.subdeck.replace(/(&nbsp;)/gim, ' ')
      this.story.authorName = this.story.authorName.replace(/(<br>)/gim, '')
      this.story.authorName = this.story.authorName.replace(/(&nbsp;)/gim, ' ')
      this.story.authorTitle = this.story.authorTitle.replace(/(<br>)/gim, '')
      this.story.authorTitle = this.story.authorTitle.replace(/(&nbsp;)/gim, ' ')
      this.story.bodyText = this.story.bodyText.replace(/(&nbsp;)/gim, ' ')

      // remove snippet object from story object, then POST
      let copy = Object.assign({}, this.story)
      copy.snippet = null
      fetch('http://127.0.0.1:8000/validate-story', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(copy)
      }).then(response => {
        return response.json()
      }).then(resp => {
        this.validationErrors = resp
        this.didValidation = true
      })
    },
    editKicker (operation) {
      this.story.kicker = operation.api.origElements.innerHTML
      this.didValidation = false
    },
    editHeadline (operation) {
      this.story.headline = operation.api.origElements.innerHTML
      this.didValidation = false
    },
    editSubdeck (operation) {
      this.story.subdeck = operation.api.origElements.innerHTML
      this.didValidation = false
    },
    editAuthorName (operation) {
      this.story.authorName = operation.api.origElements.innerHTML
      this.didValidation = false
    },
    editAuthorTitle (operation) {
      this.story.authorTitle = operation.api.origElements.innerHTML
      this.didValidation = false
    },
    editBodyText (operation) {
      this.story.bodyText = operation.api.origElements.innerHTML
      this.didValidation = false
    }
  },
  props: [
    'story'
  ],
  components: {
    'medium-editor': editor,
    FontAwesomeIcon
  },
  computed: {
    canCreatePost () {
      if (!this.didValidation) return false
      if (this.validationErrors.length === 0) return true
      return false
    },
    syncIcon () {
      return faCheck
    },
    uploadIcon () {
      return faCloudUploadAlt
    }
  }
}
</script>

<style scoped>
.medium-editor-element {
  outline: none;
  white-space: pre-wrap;
}
.medium-editor-element >>> p {
  margin-top: 20px;
}
.author-name {
  min-height: 0;
}
.author-title {
  min-height: 0;
}
ul.validation-errors {
  list-style-type: none;
}
ul.validation-errors > li {
  text-indent: 5px;
}
ul.validation-errors > li:before {
  content: "- ";
  text-indent: -5px;
}
</style>
