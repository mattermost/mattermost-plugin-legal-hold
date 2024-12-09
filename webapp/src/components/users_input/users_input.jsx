// This component is taken from mattermost-plugin-custom-attributes

import React from 'react';
import PropTypes from 'prop-types';

import debounce from 'lodash/debounce';
import AsyncSelect from 'react-select/async';

// UsersInput searches and selects user profiles displayed by username.
// Users prop can handle the user profile object or strings directly if the user object is not available.
// Returns the selected users ids in the `OnChange` value parameter.
export default class UsersInput extends React.Component {
    static propTypes = {
        placeholder: PropTypes.string,
        users: PropTypes.array,
        onChange: PropTypes.func,
        actions: PropTypes.shape({
            searchProfiles: PropTypes.func.isRequired,
        }).isRequired,
    };

    onChange = (value) => {
        if (this.props.onChange) {
            this.props.onChange(value);
        }
    };

    getOptionValue = (user) => {
        if (user.id) {
            return user.id;
        }

        return user;
    };

    formatOptionLabel = (option) => {
        if (option.first_name && option.last_name && option.username) {
            return (
                <React.Fragment>
                    {`@${option.username} (${option.first_name} ${option.last_name})`}
                </React.Fragment>
            );
        }

        if (option.username) {
            return (
                <React.Fragment>
                    {`@${option.username}`}
                </React.Fragment>
            );
        }

        return option;
    };

    debouncedSearchProfiles = debounce((term, callback) => {
        this.props.actions.searchProfiles(term, {allow_inactive: true}).then(({data}) => {
            callback(data);
        }).catch(() => {
            // eslint-disable-next-line no-console
            console.error('Error searching user profiles in custom attribute settings dropdown.');
            callback([]);
        });
    }, 150);

    usersLoader = (term, callback) => {
        try {
            this.debouncedSearchProfiles(term, callback);
        } catch (error) {
            // eslint-disable-next-line no-console
            console.error(error);
            callback([]);
        }
    };

    keyDownHandler = (e) => {
        if (e.key === 'Enter') {
            e.stopPropagation();
        }
    };

    render() {
        return (
            <AsyncSelect
                isMulti={true}
                cacheOptions={true}
                defaultOptions={false}
                loadOptions={this.usersLoader}
                onChange={this.onChange}
                getOptionValue={this.getOptionValue}
                formatOptionLabel={this.formatOptionLabel}
                defaultMenuIsOpen={false}
                openMenuOnClick={false}
                isClearable={false}
                placeholder={this.props.placeholder}
                value={this.props.users}
                components={{DropdownIndicator: () => null, IndicatorSeparator: () => null}}
                styles={customStyles}
                menuPortalTarget={document.body}
                menuPosition={'fixed'}
                onKeyDown={this.keyDownHandler}
            />
        );
    }
}

const customStyles = {
    container: (base) => ({
        ...base,
    }),
    control: (base) => ({
        ...base,
        minHeight: '46px',
    }),
    menuPortal: (base) => ({
        ...base,
        zIndex: 9999,
    }),
    multiValue: (base) => ({
        ...base,
        borderRadius: '50px',
    }),
};
