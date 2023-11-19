import {connect} from 'react-redux';
import {AnyAction, bindActionCreators, Dispatch} from 'redux';

import {getMissingProfilesByIds} from "mattermost-redux/actions/users";

import LegalHoldTable from '@/components/legal_hold_table/legal_hold_table';

function mapDispatchToProps(dispatch: Dispatch<AnyAction>) {
    return {
        actions: bindActionCreators({
            getMissingProfilesByIds,
        }, dispatch),
    };
}

export default connect(null, mapDispatchToProps)(LegalHoldTable);
