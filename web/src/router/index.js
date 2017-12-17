import Vue from 'vue'
import Router from 'vue-router'
import PostEditor from '@/components/PostEditor'
import PostList from '@/components/PostList'

Vue.use(Router)

export default new Router({
  mode: 'history',
  routes: [
    {
      path: '/',
      name: 'PostList',
      component: PostList
    },
    {
      path: '/editor',
      name: 'PostEditor',
      component: PostEditor,
      props: true
    }
  ]
})
