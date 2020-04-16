import miniGraph from '../../components/minigraph'
import capacityMap from '../../components/capacitymap'
import nodesTable from '../../components/nodestable'
import scrollablecard from '../../components/scrollablecard'
import { mapGetters, mapActions } from 'vuex'

export default {
  name: 'capacity',
  components: { miniGraph, capacityMap, nodesTable, scrollablecard },
  props: [],
  data () {
    return {
      showDialog: false,
      dilogTitle: 'title',
      dialogBody: '',
      dialogActions: [],
      dialogImage: null,
      block: null,
      showBadge: true,
      menu: false,
      selectedNode: ''
    }
  },
  computed: {
    ...mapGetters([
      'nodeSpecs',
      'registeredNodes',
      'page'
    ])
  },
  mounted () {
    this.getRegisteredNodes({ size: 100, page: 1 })
    this.getRegisteredFarms()
    this.initialiseNodesLoading()
    // this.initialiseRefresh()
  },

  methods: {
    ...mapActions(['getRegisteredNodes', 'getRegisteredFarms']),
    changeSelectedNode (data) {
      this.selectedNode = data
    },
    initialiseRefresh () {
      const that = this
      this.refreshInterval = setInterval(() => {
        that.getRegisteredNodes()
        that.getRegisteredFarms()
      }, 60000) // refresh every 10 minutes
    },
    initialiseNodesLoading () {
      const that = this
      this.nodesLoadingInterval = setInterval(() => {
        that.getRegisteredNodes({ size: 50, page: undefined })
      }, 500)
    }
  }
}
