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
      'registeredNodes'
    ])
  },
  mounted () {
    this.refreshData()
  },

  methods: {
    ...mapActions(['refreshData']),
    changeSelectedNode (data) {
      this.selectedNode = data
    }
  }
}
