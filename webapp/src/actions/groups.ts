import type {Group} from '@mattermost/types/groups';

import GroupTypes from 'mattermost-redux/action_types/groups';
import type {ActionFuncAsync} from 'mattermost-redux/types/actions';

import Client from '@/client';

// Requests that are in flight. A second caller waits for the same request, so it also sees the same failure.
// A request resolves to null when the group does not exist.
const pendingGroupRequests = new Map<string, Promise<Group | null>>();

const requestGroup = (id: string) => {
    let request = pendingGroupRequests.get(id);
    if (!request) {
        request = (Client.getGroup(id) as Promise<Group>).
            catch((error) => {
                if (error?.status_code === 404) {
                    return null;
                }
                throw error;
            }).
            finally(() => pendingGroupRequests.delete(id));
        pendingGroupRequests.set(id, request);
    }
    return request;
};

// Loads the groups that the store does not hold. A group that does not exist is skipped.
// A failed request does not stop the other groups from loading. The result has an `error` when a request failed.
export function getMissingGroupsByIds(groupIds: string[]): ActionFuncAsync<Group[]> {
    return async (dispatch, getState) => {
        const {groups} = getState().entities.groups;
        const missingIds = groupIds.filter((id) => !groups[id]);

        if (missingIds.length === 0) {
            return {data: []};
        }

        const results = await Promise.allSettled(missingIds.map(requestGroup));
        const fetchedGroups = results.
            filter((result): result is PromiseFulfilledResult<Group> => result.status === 'fulfilled' && result.value !== null).
            map((result) => result.value);
        const failed = results.find((result): result is PromiseRejectedResult => result.status === 'rejected');

        if (fetchedGroups.length > 0) {
            dispatch({
                type: GroupTypes.RECEIVED_GROUPS,
                data: fetchedGroups,
            });
        }

        if (failed) {
            console.log(failed.reason); //eslint-disable-line no-console
            return {error: failed.reason};
        }

        return {data: fetchedGroups};
    };
}
