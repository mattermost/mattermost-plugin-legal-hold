import {connect} from 'react-redux';
import {AnyAction, bindActionCreators, Dispatch} from 'redux';

import {logError} from 'mattermost-redux/actions/errors';
import {getMissingProfilesByIds} from 'mattermost-redux/actions/users';
import {forceLogoutIfNecessary} from 'mattermost-redux/actions/helpers';
import {ActionFunc, DispatchFunc, GetStateFunc} from 'mattermost-redux/types/actions';

import GroupTypes from 'mattermost-redux/action_types/groups';

import Client from '@/client';
import LegalHoldTable from '@/components/legal_hold_table/legal_hold_table';

function mapDispatchToProps(dispatch: Dispatch<AnyAction>) {
    return {
        actions: bindActionCreators({
            getMissingProfilesByIds,
            getMissingGroupsByIds,
        }, dispatch),
    };
}

// keep track of ongoing requests to ensure we don't try
// to query for the same groups simultaneously
const pendingGroupRequests = new Set<string>();

export function getMissingGroupsByIds(groupIds: string[]): ActionFunc {
    return async (dispatch: DispatchFunc, getState: GetStateFunc) => {
        const state = getState();
        const {groups} = state.entities.groups;
        const missingIds: string[] = [];

        groupIds.forEach((id) => {
            if (!groups[id] && !pendingGroupRequests.has(id)) {
                missingIds.push(id);
            }
        });

        if (missingIds.length == 0) {
            return {data: []};
        }

        missingIds.forEach(id => pendingGroupRequests.add(id));

        const fetchedGroups = [];
        let lastError = null;
        
        for (const groupId of missingIds) {
            try {
                const group = await Client.getGroup(groupId);
                fetchedGroups.push(group);
            } catch (error) {
                forceLogoutIfNecessary(error, dispatch, getState);
                dispatch(logError(error));
                lastError = error;
            }
        }

        missingIds.forEach(id => pendingGroupRequests.delete(id));

        if (fetchedGroups.length > 0) {
            dispatch({
                type: GroupTypes.RECEIVED_GROUPS,
                data: fetchedGroups,
            });
            return {data: fetchedGroups};
        }

	if (lastError) {
	    return {error: lastError};
	}

        return {data: []};
    };
}

export default connect(null, mapDispatchToProps)(LegalHoldTable);
