import Vue from 'vue'
import Router from 'vue-router'
import PostEditor from '@/components/PostEditor'

Vue.use(Router)

export default new Router({
  mode: 'history',
  routes: [
    {
      path: '/',
      name: 'PostEditor',
      component: PostEditor
    }
  ]
})
