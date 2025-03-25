import React from 'react';
import PropTypes from 'prop-types';
import debounce from 'lodash/debounce';
import AsyncSelect from 'react-select/async';

export default class GroupsInput extends React.Component {
    static propTypes = {
        placeholder: PropTypes.string,
        groups: PropTypes.array,
        onChange: PropTypes.func,
        actions: PropTypes.shape({
            searchGroups: PropTypes.func.isRequired,
        }).isRequired,
    };

    onChange = (value) => {
        if (this.props.onChange) {
            this.props.onChange(value);
        }
    };

    getOptionValue = (group) => {
        if (group.id) {
            return group.id;
        }
        return group;
    };

    formatOptionLabel = (option) => {
        if (option.display_name) {
            return (
                <React.Fragment>
                    {option.display_name}
                </React.Fragment>
            );
        }
        return option;
    };

    debouncedSearchGroups = debounce((term, callback) => {
        this.props.actions.searchGroups(term).then((data) => {
            callback(data);
        }).catch(() => {
            // eslint-disable-next-line no-console
            console.error('Error searching groups in legal hold settings dropdown.');
            callback([]);
        });
    }, 150);

    groupsLoader = (term, callback) => {
        try {
            this.debouncedSearchGroups(term, callback);
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
                loadOptions={this.groupsLoader}
                onChange={this.onChange}
                getOptionValue={this.getOptionValue}
                formatOptionLabel={this.formatOptionLabel}
                defaultMenuIsOpen={false}
                openMenuOnClick={false}
                isClearable={false}
                placeholder={this.props.placeholder}
                value={this.props.groups}
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
