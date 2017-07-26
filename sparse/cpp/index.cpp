#include "index.h"

#include <map>
#include <vector>

struct Chunk {
    int64_t disk_start;
    int64_t length;
};

typedef std::map<int64_t, Chunk, std::greater<int64_t>> Map;

struct Index {
    Map data;
};

Index* sparse_create() {
    return new Index;
}

void sparse_free(Index* index) {
    delete index;
}

size_t sparse_iterator_size() {
    return sizeof(Map::const_iterator);
}

void sparse_begin(const Index* index, void* iterator) {
    Map::const_iterator& it = *reinterpret_cast<Map::const_iterator*>(iterator);
    it = index->data.begin();
}

int sparse_read_chunk(const Index* index, void* iterator, int64_t* start, int64_t* disk_start, int64_t* chunk_length) {
    Map::const_iterator& it = *reinterpret_cast<Map::const_iterator*>(iterator);
    if (it == index->data.end()) {
        return 0;
    }
    it++;
    *start = it->first;
    *disk_start = it->second.disk_start;
    *chunk_length = it->second.length;
    return 1;
}

int64_t sparse_read(const Index* index, int64_t start, int64_t* disk_start, int64_t* slice_length) {
    auto it = index->data.lower_bound(start);
    if (it == index->data.end()) {
        *slice_length = 0;
    } else {
        auto offset = start - it->first;
        if (offset >= it->second.length) {
            *slice_length = 0;
        } else {
            *disk_start = it->second.disk_start + offset;
            *slice_length = it->second.length - offset;
        }
    }
    if (it == index->data.begin()) {
        return -1;
    }
    it--;
    return it->first - (start + *slice_length);
}

void sparse_write(Index* index, int64_t start, int64_t disk_start, int64_t slice_length) {
    // Find all existing chunks that may overlap with the new slice.
    auto it = index->data.lower_bound(start);
    std::vector<Map::iterator> iterators;
    if (it != index->data.end()) {
        iterators.push_back(it);
        while (it != index->data.begin()) {
            it--;
            if (it->first >= start + slice_length) {
                break;
            }
            iterators.push_back(it);
        }
    }
    int64_t slice_begin = start;
    int64_t slice_end = start + slice_length;
    for (auto it : iterators) {
        int64_t chunk_begin = it->first;
        int64_t chunk_end = it->first + it->second.length;
        if (slice_end < chunk_begin || chunk_end < slice_begin) {
            // No overlap.
            continue;
        }
        bool left_outside = chunk_begin < slice_begin;
        bool right_outside = slice_end < chunk_end;
        if (!left_outside && !right_outside) {
            // slice: ******
            // chunk:  ----
            index->data.erase(it);
        } else if (left_outside && right_outside) {
            // slice:  ****
            // chunk: ------
            int64_t disk_start = it->second.disk_start;
            // Reuse it as new left as they have the same start.
            Chunk& left = it->second;
            left.length = slice_begin - chunk_begin;
            int64_t delta_to_right = slice_end - chunk_begin;
            Chunk right;
            right.length = chunk_end - slice_end;
            right.disk_start = disk_start + delta_to_right;
            int64_t right_start = slice_end;
            index->data[right_start] = right;
        } else if (left_outside && !right_outside) {
            // slice:  ****
            // chunk: ---
            //        A
            it->second.length = slice_begin - chunk_begin;
        } else if (!left_outside && right_outside) {
            // slice: *****
            // chunk:   ----
            //             C
            int64_t delta = slice_end - chunk_begin;
            Chunk right;
            right.disk_start = it->second.disk_start + delta;
            right.length = it->second.length - delta;
            int64_t right_start = it->first + delta;
            index->data[right_start] = right;
            index->data.erase(it);
        }
    }
    // Put the slice into the map.
    Chunk slice;
    slice.disk_start = disk_start;
    slice.length = slice_length;
    auto res = index->data.insert(Map::value_type(slice_begin, slice));
    auto it2 = res.first;
    // Merge with left neighbour if possible.
    // Right neighbour can not be merged with because its disk_start is lower.
    auto left = it2;
    left++;
    if (left != index->data.end()) {
        int64_t delta = it2->first - left->first;
        if (left->second.length == delta && left->second.disk_start + delta == it2->second.disk_start) {
            // Expand left, kill it2.
            left->second.length += it2->second.length;
            index->data.erase(it2);
        }
    }
}
